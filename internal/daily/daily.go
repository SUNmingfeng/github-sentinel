// internal/daily/daily.go
package daily

import (
	"context"
	"fmt"
	"github.com/SUNmingfeng/github-sentinel/internal/github"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DailyProgress struct {
	githubClient *github.Client
	outputDir    string
}

func NewDailyProgress(client *github.Client, outputDir string) *DailyProgress {
	return &DailyProgress{
		githubClient: client,
		outputDir:    outputDir,
	}
}

// GenerateDailyReport 生成每日进展报告
func (d *DailyProgress) GenerateDailyReport(ctx context.Context, repoFullName string) error {
	owner, repo := parseRepoFullName(repoFullName)

	// 只获取最近30天的 Issues 和 PRs（避免 API 限制）
	days := 1
	issues, err := d.githubClient.GetRecentIssues(ctx, owner, repo, days)
	if err != nil {
		return fmt.Errorf("failed to get recent issues: %w", err)
	}

	pulls, err := d.githubClient.GetRecentPullRequests(ctx, owner, repo, days)
	if err != nil {
		return fmt.Errorf("failed to get recent pull requests: %w", err)
	}

	// 生成 Markdown 内容
	content := d.formatMarkdown(repoFullName, issues, pulls, days)

	// 保存文件
	filename := fmt.Sprintf("%s_%s.md",
		strings.ReplaceAll(repoFullName, "/", "_"),
		time.Now().Format("20060102"))

	filepath := filepath.Join(d.outputDir, filename)
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✓ Daily progress saved to %s (last %d days)\n", filepath, days)
	return nil
}

// formatMarkdown 格式化 Markdown 输出
// formatMarkdown 格式化 Markdown 输出（添加时间范围说明）
func (d *DailyProgress) formatMarkdown(repoName string, issues []github.IssueInfo, pulls []github.PullRequestInfo, days int) string {
	var sb strings.Builder

	// 头部
	sb.WriteString(fmt.Sprintf("# 📊 Daily Progress Report: %s\n\n", repoName))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Time Range:** Last %d days\n", days))
	sb.WriteString("---\n\n")

	// 统计摘要
	openIssues := countIssuesByState(issues, "open")
	closedIssues := countIssuesByState(issues, "closed")
	openPRs := countPRsByState(pulls, "open")
	mergedPRs := countMergedPRs(pulls)

	sb.WriteString("## 📈 Summary\n\n")
	sb.WriteString("| Type | Open | Closed/ Merged |\n")
	sb.WriteString("|------|------|----------------|\n")
	sb.WriteString(fmt.Sprintf("| Issues | %d | %d |\n", openIssues, closedIssues))
	sb.WriteString(fmt.Sprintf("| Pull Requests | %d | %d |\n\n", openPRs, mergedPRs))

	// 开放的 Issues（限制显示数量，避免报告过长）
	sb.WriteString("## 🐛 Open Issues\n\n")
	openIssuesList := filterIssuesByState(issues, "open")
	if len(openIssuesList) > 0 {
		// 只显示前20个
		displayCount := min(len(openIssuesList), 20)
		for i := 0; i < displayCount; i++ {
			issue := openIssuesList[i]
			sb.WriteString(fmt.Sprintf("- #%d: **%s** by @%s [%s]\n",
				issue.Number, issue.Title, issue.Author, strings.Join(issue.Labels, ", ")))
		}
		if len(openIssuesList) > 20 {
			sb.WriteString(fmt.Sprintf("\n_... and %d more open issues_\n", len(openIssuesList)-20))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("_No open issues at this time._\n\n")
	}

	// 开放的 PRs
	sb.WriteString("## 🔀 Open Pull Requests\n\n")
	openPRsList := filterPRsByState(pulls, "open")
	if len(openPRsList) > 0 {
		displayCount := min(len(openPRsList), 10)
		for i := 0; i < displayCount; i++ {
			pr := openPRsList[i]
			sb.WriteString(fmt.Sprintf("- #%d: **%s** by @%s (+%d/-%d)\n",
				pr.Number, pr.Title, pr.Author, pr.Additions, pr.Deletions))
		}
		if len(openPRsList) > 10 {
			sb.WriteString(fmt.Sprintf("\n_... and %d more open PRs_\n", len(openPRsList)-10))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("_No open pull requests at this time._\n\n")
	}

	// 最近合并的 PRs
	sb.WriteString("## ✅ Recently Merged\n\n")
	mergedPRsList := filterMergedPRs(pulls, 7)
	if len(mergedPRsList) > 0 {
		for _, pr := range mergedPRsList {
			sb.WriteString(fmt.Sprintf("- #%d: **%s** by @%s\n", pr.Number, pr.Title, pr.Author))
		}
	} else {
		sb.WriteString("_No recent merges._\n")
	}

	// 添加统计脚注
	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("*Report shows activity from the last %d days. For full history, please visit the GitHub repository.*\n", days))

	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 辅助函数
func parseRepoFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullName
}

func countIssuesByState(issues []github.IssueInfo, state string) int {
	count := 0
	for _, issue := range issues {
		if issue.State == state {
			count++
		}
	}
	return count
}

func filterIssuesByState(issues []github.IssueInfo, state string) []github.IssueInfo {
	var result []github.IssueInfo
	for _, issue := range issues {
		if issue.State == state {
			result = append(result, issue)
		}
	}
	return result
}

func countPRsByState(pulls []github.PullRequestInfo, state string) int {
	count := 0
	for _, pr := range pulls {
		if pr.State == state && !pr.Merged {
			count++
		}
	}
	return count
}

func filterPRsByState(pulls []github.PullRequestInfo, state string) []github.PullRequestInfo {
	var result []github.PullRequestInfo
	for _, pr := range pulls {
		if pr.State == state && !pr.Merged {
			result = append(result, pr)
		}
	}
	return result
}

func countMergedPRs(pulls []github.PullRequestInfo) int {
	count := 0
	for _, pr := range pulls {
		if pr.Merged {
			count++
		}
	}
	return count
}

func filterMergedPRs(pulls []github.PullRequestInfo, days int) []github.PullRequestInfo {
	var result []github.PullRequestInfo
	since := time.Now().AddDate(0, 0, -days)
	for _, pr := range pulls {
		if pr.Merged && pr.MergedAt != nil && pr.MergedAt.After(since) {
			result = append(result, pr)
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

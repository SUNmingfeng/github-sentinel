// internal/github/client.go
package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	ctx    context.Context
}

type Config struct {
	Token   string
	BaseURL string
}

type CommitInfo struct {
	SHA     string
	Message string
	URL     string
	Author  string
	Date    time.Time
}

type ReleaseInfo struct {
	TagName     string
	Name        string
	Body        string
	URL         string
	PublishedAt time.Time
	IsLatest    bool
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*ReleaseInfo, error) {
	release, resp, err := c.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	defer resp.Body.Close()

	return &ReleaseInfo{
		TagName:     release.GetTagName(),
		Name:        release.GetName(),
		Body:        release.GetBody(),
		URL:         release.GetHTMLURL(),
		PublishedAt: release.GetPublishedAt().Time,
		IsLatest:    true,
	}, nil
}

func (c *Client) GetRecentCommits(ctx context.Context, owner, repo string, days int) ([]CommitInfo, error) {
	opts := &github.CommitsListOptions{
		Since: time.Now().AddDate(0, 0, -days).UTC(),
	}

	commits, resp, err := c.client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	defer resp.Body.Close()

	var result []CommitInfo
	for _, commit := range commits {
		if commit.Commit != nil && commit.Commit.Author != nil {
			result = append(result, CommitInfo{
				SHA:     commit.GetSHA(),
				Message: commit.Commit.GetMessage(),
				URL:     commit.GetHTMLURL(),
				Author:  commit.Commit.Author.GetName(),
				Date:    commit.Commit.Author.GetDate().Time,
			})
		}
	}

	return result, nil
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repoObj, resp, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	defer resp.Body.Close()

	return repoObj, nil
}

func NewClient(cfg Config) *Client {
	var httpClient *http.Client
	log.Println("打印token！！！！")
	log.Println(cfg.Token)
	log.Println("打印token！！！！")
	if cfg.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.Token},
		)

		httpClient = oauth2.NewClient(context.Background(), ts)
	} else {
		httpClient = &http.Client{}
	}

	// 自定义 Transport 设置 User-Agent
	transport := &userAgentTransport{
		base:      httpClient.Transport,
		userAgent: "GitHub-Sentinel/1.0",
	}
	httpClient.Transport = transport

	ctx := context.Background()
	client := github.NewClient(httpClient)
	client.UserAgent = "github_sentinel_app"

	return &Client{
		client: client,
		ctx:    ctx,
	}
}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	return t.base.RoundTrip(req)
}

type IssueInfo struct {
	Number        int        `json:"number"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	URL           string     `json:"url"`
	Author        string     `json:"author"`
	State         string     `json:"state"`    // open, closed
	Labels        []string   `json:"labels"`   // 标签列表
	Comments      int        `json:"comments"` // 评论数
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	IsPullRequest bool       `json:"is_pull_request"`
}

// PullRequestInfo 完整定义
type PullRequestInfo struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	URL          string     `json:"url"`
	Author       string     `json:"author"`
	State        string     `json:"state"`
	Mergeable    bool       `json:"mergeable"`
	Merged       bool       `json:"merged"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
}

// GetAllIssues 获取仓库的所有 Issues（使用基于游标的分页）
func (c *Client) GetAllIssues(ctx context.Context, owner, repo string, state string) ([]IssueInfo, error) {
	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100, // 每页最大数量
		},
	}

	var allIssues []IssueInfo
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			// 检查是否是 422 错误（不支持 page 参数）
			if resp != nil && resp.StatusCode == 422 {
				// 回退到基于时间范围的分页
				return c.getAllIssuesWithTimeRange(ctx, owner, repo, state)
			}
			return nil, fmt.Errorf("failed to get issues: %w", err)
		}

		for _, issue := range issues {
			// 过滤掉 Pull Requests
			if issue.IsPullRequest() {
				continue
			}

			labels := make([]string, 0)
			for _, label := range issue.Labels {
				if label.Name != nil {
					labels = append(labels, *label.Name)
				}
			}

			allIssues = append(allIssues, IssueInfo{
				Number:        issue.GetNumber(),
				Title:         issue.GetTitle(),
				Body:          issue.GetBody(),
				URL:           issue.GetHTMLURL(),
				Author:        issue.GetUser().GetLogin(),
				State:         issue.GetState(),
				Labels:        labels,
				Comments:      issue.GetComments(),
				CreatedAt:     issue.GetCreatedAt().Time,
				UpdatedAt:     issue.GetUpdatedAt().Time,
				ClosedAt:      issue.ClosedAt.GetTime(),
				IsPullRequest: issue.IsPullRequest(),
			})
		}

		// 检查是否有下一页（使用 NextPage）
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// getAllIssuesWithTimeRange 使用时间范围获取 issues（回退方案）
func (c *Client) getAllIssuesWithTimeRange(ctx context.Context, owner, repo string, state string) ([]IssueInfo, error) {
	var allIssues []IssueInfo

	// 使用 Since 参数获取最近的数据
	// 如果仓库太大，只获取最近3个月的数据
	since := time.Now().AddDate(0, -3, 0)

	opts := &github.IssueListByRepoOptions{
		State: state,
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get issues with time range: %w", err)
		}

		for _, issue := range issues {
			if issue.IsPullRequest() {
				continue
			}

			labels := make([]string, 0)
			for _, label := range issue.Labels {
				if label.Name != nil {
					labels = append(labels, *label.Name)
				}
			}

			allIssues = append(allIssues, IssueInfo{
				Number:        issue.GetNumber(),
				Title:         issue.GetTitle(),
				Body:          issue.GetBody(),
				URL:           issue.GetHTMLURL(),
				Author:        issue.GetUser().GetLogin(),
				State:         issue.GetState(),
				Labels:        labels,
				Comments:      issue.GetComments(),
				CreatedAt:     issue.GetCreatedAt().Time,
				UpdatedAt:     issue.GetUpdatedAt().Time,
				ClosedAt:      issue.ClosedAt.GetTime(),
				IsPullRequest: issue.IsPullRequest(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// GetAllPullRequests 获取仓库的所有 Pull Requests（使用基于游标的分页）
func (c *Client) GetAllPullRequests(ctx context.Context, owner, repo string, state string) ([]PullRequestInfo, error) {
	opts := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allPRs []PullRequestInfo
	for {
		pulls, resp, err := c.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			// 如果使用 page 参数失败，使用时间范围
			if resp != nil && resp.StatusCode == 422 {
				return c.getAllPRsWithTimeRange(ctx, owner, repo, state)
			}
			return nil, fmt.Errorf("failed to get pull requests: %w", err)
		}

		for _, pull := range pulls {
			allPRs = append(allPRs, PullRequestInfo{
				Number:       pull.GetNumber(),
				Title:        pull.GetTitle(),
				Body:         pull.GetBody(),
				URL:          pull.GetHTMLURL(),
				Author:       pull.GetUser().GetLogin(),
				State:        pull.GetState(),
				Mergeable:    pull.GetMergeable(),
				Merged:       pull.GetMerged(),
				Additions:    pull.GetAdditions(),
				Deletions:    pull.GetDeletions(),
				ChangedFiles: pull.GetChangedFiles(),
				CreatedAt:    pull.GetCreatedAt().Time,
				UpdatedAt:    pull.GetUpdatedAt().Time,
				MergedAt:     pull.MergedAt.GetTime(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// getAllPRsWithTimeRange 使用时间范围获取 PRs（回退方案）
func (c *Client) getAllPRsWithTimeRange(ctx context.Context, owner, repo string, state string) ([]PullRequestInfo, error) {
	var allPRs []PullRequestInfo

	// 使用 Since 参数获取最近的数据
	since := time.Now().AddDate(0, -3, 0)

	opts := &github.PullRequestListOptions{
		State:     state,
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// 注意：PullRequests API 不支持 Since 参数，需要手动过滤
	pulls, _, err := c.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	for _, pull := range pulls {
		// 只获取最近3个月的 PRs
		if pull.GetCreatedAt().Time.After(since) {
			allPRs = append(allPRs, PullRequestInfo{
				Number:       pull.GetNumber(),
				Title:        pull.GetTitle(),
				Body:         pull.GetBody(),
				URL:          pull.GetHTMLURL(),
				Author:       pull.GetUser().GetLogin(),
				State:        pull.GetState(),
				Mergeable:    pull.GetMergeable(),
				Merged:       pull.GetMerged(),
				Additions:    pull.GetAdditions(),
				Deletions:    pull.GetDeletions(),
				ChangedFiles: pull.GetChangedFiles(),
				CreatedAt:    pull.GetCreatedAt().Time,
				UpdatedAt:    pull.GetUpdatedAt().Time,
				MergedAt:     pull.MergedAt.GetTime(),
			})
		}
	}

	return allPRs, nil
}

// GetRecentIssues 获取最近的 Issues（推荐用于大型仓库）
func (c *Client) GetRecentIssues(ctx context.Context, owner, repo string, days int) ([]IssueInfo, error) {
	since := time.Now().AddDate(0, 0, -days)

	opts := &github.IssueListByRepoOptions{
		State: "all",
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []IssueInfo
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent issues: %w", err)
		}

		for _, issue := range issues {
			if issue.IsPullRequest() {
				continue
			}

			labels := make([]string, 0)
			for _, label := range issue.Labels {
				if label.Name != nil {
					labels = append(labels, *label.Name)
				}
			}

			allIssues = append(allIssues, IssueInfo{
				Number:        issue.GetNumber(),
				Title:         issue.GetTitle(),
				Body:          issue.GetBody(),
				URL:           issue.GetHTMLURL(),
				Author:        issue.GetUser().GetLogin(),
				State:         issue.GetState(),
				Labels:        labels,
				Comments:      issue.GetComments(),
				CreatedAt:     issue.GetCreatedAt().Time,
				UpdatedAt:     issue.GetUpdatedAt().Time,
				ClosedAt:      issue.ClosedAt.GetTime(),
				IsPullRequest: issue.IsPullRequest(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// GetRecentPullRequests 获取最近的 Pull Requests（推荐用于大型仓库）
func (c *Client) GetRecentPullRequests(ctx context.Context, owner, repo string, days int) ([]PullRequestInfo, error) {
	since := time.Now().AddDate(0, 0, -days)

	opts := &github.PullRequestListOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	pulls, _, err := c.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent pull requests: %w", err)
	}

	var allPRs []PullRequestInfo
	for _, pull := range pulls {
		if pull.GetCreatedAt().Time.After(since) {
			allPRs = append(allPRs, PullRequestInfo{
				Number:       pull.GetNumber(),
				Title:        pull.GetTitle(),
				Body:         pull.GetBody(),
				URL:          pull.GetHTMLURL(),
				Author:       pull.GetUser().GetLogin(),
				State:        pull.GetState(),
				Mergeable:    pull.GetMergeable(),
				Merged:       pull.GetMerged(),
				Additions:    pull.GetAdditions(),
				Deletions:    pull.GetDeletions(),
				ChangedFiles: pull.GetChangedFiles(),
				CreatedAt:    pull.GetCreatedAt().Time,
				UpdatedAt:    pull.GetUpdatedAt().Time,
				MergedAt:     pull.MergedAt.GetTime(),
			})
		}
	}

	return allPRs, nil
}

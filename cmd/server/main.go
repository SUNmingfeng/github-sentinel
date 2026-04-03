// cmd/server/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/SUNmingfeng/github-sentinel/internal/daily"
	"github.com/SUNmingfeng/github-sentinel/internal/llm"
	"github.com/SUNmingfeng/github-sentinel/internal/reporter"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SUNmingfeng/github-sentinel/internal/aggregator"
	"github.com/SUNmingfeng/github-sentinel/internal/github"
	"github.com/SUNmingfeng/github-sentinel/internal/scheduler"
	"github.com/SUNmingfeng/github-sentinel/internal/service"
)

type GitHubSentinel struct {
	ctx           context.Context
	cancel        context.CancelFunc
	githubClient  *github.Client
	llmClient     *llm.DeepSeekClient
	repoService   *service.RepoService
	scheduler     *scheduler.Scheduler
	aggregator    *aggregator.Aggregator
	subscriptions map[string]string // repo -> frequency
}

func NewGitHubSentinel(token string) *GitHubSentinel {
	ctx, cancel := context.WithCancel(context.Background())

	githubClient := github.NewClient(github.Config{
		Token:   token,
		BaseURL: "",
	})

	dpApiKey := os.Getenv("DEEPSEEK_API_KEY")
	cfg := llm.Config{
		APIKey: dpApiKey,
		Model:  "deepseek-chat",
	}
	llmClient, err := llm.NewDeepSeekClient(cfg)
	if err != nil {
		panic(err)
	}

	repoService := service.NewRepoService(githubClient)
	sched := scheduler.NewScheduler(repoService)
	agg := aggregator.NewAggregator()

	sentinel := &GitHubSentinel{
		ctx:           ctx,
		cancel:        cancel,
		githubClient:  githubClient,
		llmClient:     llmClient,
		repoService:   repoService,
		scheduler:     sched,
		aggregator:    agg,
		subscriptions: make(map[string]string),
	}

	// 添加报告处理器
	sched.AddReportHandler(sentinel.handleScheduledReport)

	return sentinel
}

// 处理调度器生成的报告
func (gs *GitHubSentinel) handleScheduledReport(repoName string, report string, eventsCount int) {
	fmt.Printf("\n🔔 [Scheduled Report] %s\n", repoName)
	fmt.Printf("   Events: %d | Time: %s\n", eventsCount, time.Now().Format("15:04:05"))
	fmt.Printf("   Report saved to reports/ directory\n")
	fmt.Print("\n> ") // 重新显示提示符
}

func (gs *GitHubSentinel) Start() {
	// 启动调度器（在后台运行）
	go gs.scheduler.Start()

	// 启动交互式控制台
	gs.startConsole()
}

func (gs *GitHubSentinel) startConsole() {
	scanner := bufio.NewScanner(os.Stdin)

	gs.printHelp()

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "help", "h", "?":
			gs.printHelp()

		case "subscribe", "sub":
			if len(parts) < 2 {
				fmt.Println("Usage: subscribe <owner/repo> [daily|weekly]")
				continue
			}
			repoName := parts[1]
			frequency := "daily"
			if len(parts) >= 3 {
				frequency = parts[2]
			}
			gs.subscribe(repoName, frequency)

		case "unsubscribe", "unsub":
			if len(parts) < 2 {
				fmt.Println("Usage: unsubscribe <owner/repo>")
				continue
			}
			repoName := parts[1]
			gs.unsubscribe(repoName)

		case "list", "ls":
			gs.listSubscriptions()

		case "fetch", "get":
			if len(parts) < 2 {
				fmt.Println("Usage: fetch <owner/repo> [days]")
				continue
			}
			repoName := parts[1]
			days := 7
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &days)
			}
			gs.fetchNow(repoName, days)

		case "fetch-all":
			days := 7
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &days)
			}
			gs.fetchAllNow(days)

		case "status", "stat":
			gs.showStatus()

		case "clear":
			gs.clearScreen()

		case "quit", "exit", "q":
			fmt.Println("Shutting down GitHub Sentinel...")
			gs.cancel()
			gs.scheduler.Stop()
			os.Exit(0)

		case "daily", "progress":
			if len(parts) < 2 {
				fmt.Println("Usage: daily <owner/repo>")
				continue
			}
			repoName := parts[1]
			gs.generateDailyProgress(repoName)

		case "aireport", "ai":
			if len(parts) < 2 {
				fmt.Println("Usage: aireport <owner/repo>")
				continue
			}
			repoName := parts[1]
			gs.generateAIReport(repoName)

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			gs.printHelp()
		}
	}
}

func (gs *GitHubSentinel) subscribe(repoName, frequency string) {
	if _, exists := gs.subscriptions[repoName]; exists {
		fmt.Printf("Already subscribed to %s\n", repoName)
		return
	}

	// 验证仓库是否存在
	owner, repo := parseRepoFullName(repoName)
	_, err := gs.githubClient.GetRepository(gs.ctx, owner, repo)
	if err != nil {
		fmt.Printf("Error: Repository %s not found or inaccessible: %v\n", repoName, err)
		return
	}

	// 添加到调度器
	switch frequency {
	case "daily":
		if err := gs.scheduler.AddDailyJob(repoName); err != nil {
			fmt.Printf("Failed to add daily job: %v\n", err)
			return
		}
	case "weekly":
		if err := gs.scheduler.AddWeeklyJob(repoName); err != nil {
			fmt.Printf("Failed to add weekly job: %v\n", err)
			return
		}
	default:
		fmt.Printf("Invalid frequency: %s. Use 'daily' or 'weekly'\n", frequency)
		return
	}

	// 添加到订阅列表
	gs.subscriptions[repoName] = frequency
	fmt.Printf("✓ Subscribed to %s (%s updates)\n", repoName, frequency)

	// 可选：立即生成每日进展
	fmt.Printf("✓ Subscribed to %s (%s updates)\n", repoName, frequency)
	fmt.Print("Generate daily progress now? (y/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(scanner.Text()) == "y" {
		gs.generateDailyProgress(repoName)
	}
}

func (gs *GitHubSentinel) unsubscribe(repoName string) {
	if _, exists := gs.subscriptions[repoName]; !exists {
		fmt.Printf("Not subscribed to %s\n", repoName)
		return
	}

	delete(gs.subscriptions, repoName)
	gs.scheduler.RemoveJob(repoName)
	fmt.Printf("✓ Unsubscribed from %s\n", repoName)
}

func (gs *GitHubSentinel) listSubscriptions() {
	if len(gs.subscriptions) == 0 {
		fmt.Println("No active subscriptions")
		return
	}

	fmt.Println("\nActive subscriptions:")
	fmt.Println("--------------------")
	for repo, freq := range gs.subscriptions {
		fmt.Printf("  • %s (%s)\n", repo, freq)
	}
	fmt.Printf("\nTotal: %d subscriptions\n", len(gs.subscriptions))
}

func (gs *GitHubSentinel) fetchNow(repoName string, days int) {
	fmt.Printf("Fetching updates for %s (last %d days)...\n", repoName, days)

	events, release, err := gs.repoService.FetchRepoUpdates(gs.ctx, repoName, days)
	if err != nil {
		fmt.Printf("Error fetching updates: %v\n", err)
		return
	}

	// 生成报告
	report := gs.aggregator.GenerateReport(repoName, release, events)

	// 输出到控制台
	fmt.Println("\n" + report)

	// 保存到文件
	reportsDir := "reports"
	os.MkdirAll(reportsDir, 0755)
	filename := fmt.Sprintf("%s/%s_%s.md",
		reportsDir,
		strings.ReplaceAll(repoName, "/", "_"),
		time.Now().Format("20060102_150405"))

	if err := os.WriteFile(filename, []byte(report), 0644); err != nil {
		fmt.Printf("Failed to save report: %v\n", err)
	} else {
		fmt.Printf("\n✓ Report saved to %s\n", filename)
	}
}

func (gs *GitHubSentinel) fetchAllNow(days int) {
	if len(gs.subscriptions) == 0 {
		fmt.Println("No subscriptions to fetch")
		return
	}

	fmt.Printf("Fetching updates for all %d subscriptions (last %d days)...\n",
		len(gs.subscriptions), days)

	reportsDir := "reports"
	os.MkdirAll(reportsDir, 0755)

	for repoName := range gs.subscriptions {
		fmt.Printf("\n--- %s ---\n", repoName)
		events, release, err := gs.repoService.FetchRepoUpdates(gs.ctx, repoName, days)
		if err != nil {
			fmt.Printf("Error fetching %s: %v\n", repoName, err)
			continue
		}

		report := gs.aggregator.GenerateReport(repoName, release, events)
		fmt.Println(report)

		// 保存报告
		filename := fmt.Sprintf("%s/%s_%s.md",
			reportsDir,
			strings.ReplaceAll(repoName, "/", "_"),
			time.Now().Format("20060102_150405"))
		os.WriteFile(filename, []byte(report), 0644)
	}

	fmt.Println("\n✓ All reports generated and saved to reports/ directory")
}

func (gs *GitHubSentinel) showStatus() {
	fmt.Println("\nGitHub Sentinel Status")
	fmt.Println("=====================")
	fmt.Printf("Active Subscriptions: %d\n", len(gs.subscriptions))
	fmt.Printf("Scheduler: %s\n", gs.scheduler.GetStatus())
	fmt.Printf("GitHub API: Connected\n")
	fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	if len(gs.subscriptions) > 0 {
		fmt.Println("\nMonitored Repositories:")
		for repo, freq := range gs.subscriptions {
			fmt.Printf("  • %s (%s)\n", repo, freq)
		}
	}
}

func (gs *GitHubSentinel) clearScreen() {
	fmt.Print("\033[2J\033[H")
	gs.printHelp()
}

func (gs *GitHubSentinel) printHelp() {
	fmt.Println("\nGitHub Sentinel - Interactive Console (v0.2)")
	fmt.Println("============================================")
	fmt.Println("Commands:")
	fmt.Println("  subscribe <owner/repo> [daily|weekly]  - Subscribe to a repository")
	fmt.Println("  unsubscribe <owner/repo>               - Unsubscribe from a repository")
	fmt.Println("  list, ls                               - List all subscriptions")
	fmt.Println("  fetch <owner/repo> [days]              - Fetch updates immediately")
	fmt.Println("  fetch-all [days]                       - Fetch all subscriptions")
	fmt.Println("  daily <owner/repo>                     - Generate daily progress report")
	fmt.Println("  aireport <owner/repo>                  - Generate AI-powered report (v0.2)")
	fmt.Println("  status, stat                           - Show system status")
	fmt.Println("  clear                                  - Clear screen")
	fmt.Println("  help, h, ?                             - Show this help")
	fmt.Println("  quit, exit, q                          - Exit program")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  subscribe langchain-ai/langchain daily")
	fmt.Println("  daily langchain-ai/langchain")
	fmt.Println("  fetch langchain-ai/langchain 7")
	fmt.Println("")
	fmt.Println("v0.2 New Features:")
	fmt.Println("  • Daily progress reports with issues and PRs")
	fmt.Println("  • AI-powered summaries (requires DEEPSEEK_API_KEY)")
}

func main() {
	// 获取 GitHub Token
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		fmt.Println("Warning: GITHUB_TOKEN not set, API rate limit will be very low")
		fmt.Println("You can set it with: export GITHUB_TOKEN=your_token_here")
		fmt.Println()
	}

	// 创建 reports 目录
	os.MkdirAll("reports", 0755)

	// 创建信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建 Sentinel
	sentinel := NewGitHubSentinel(githubToken)

	// 启动 Sentinel
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived shutdown signal...")
		sentinel.cancel()
		sentinel.scheduler.Stop()
		os.Exit(0)
	}()

	fmt.Println("GitHub Sentinel - AI-Powered Repository Monitor")
	fmt.Println("Type 'help' for available commands\n")

	sentinel.Start()
}

func parseRepoFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullName
}

func (gs *GitHubSentinel) generateDailyProgress(repoName string) {
	daily := daily.NewDailyProgress(gs.githubClient, "daily")
	if err := daily.GenerateDailyReport(gs.ctx, repoName); err != nil {
		fmt.Printf("Error generating daily progress: %v\n", err)
	}
}

func (gs *GitHubSentinel) generateAIReport(repoName string) {
	if gs.llmClient == nil {
		fmt.Println("DeepSeek client not configured. Please set DEEPSEEK_API_KEY environment variable.")
		return
	}

	reporter := reporter.NewReporter(gs.githubClient, gs.llmClient, "daily", "reports")
	if err := reporter.GenerateAIDailyReport(gs.ctx, repoName); err != nil {
		fmt.Printf("Error generating AI report: %v\n", err)
	}
}

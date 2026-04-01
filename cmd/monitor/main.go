// cmd/server/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
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
	repoService   *service.RepoService
	scheduler     *scheduler.Scheduler
	aggregator    *aggregator.Aggregator
	subscriptions map[string]bool // 订阅的仓库列表
}

func NewGitHubSentinel(token string) *GitHubSentinel {
	ctx, cancel := context.WithCancel(context.Background())

	githubClient := github.NewClient(github.Config{
		Token:   token,
		BaseURL: "",
	})

	repoService := service.NewRepoService(githubClient)
	sched := scheduler.NewScheduler(repoService)
	agg := aggregator.NewAggregator()

	return &GitHubSentinel{
		ctx:           ctx,
		cancel:        cancel,
		githubClient:  githubClient,
		repoService:   repoService,
		scheduler:     sched,
		aggregator:    agg,
		subscriptions: make(map[string]bool),
	}
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

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			gs.printHelp()
		}
	}
}

func (gs *GitHubSentinel) subscribe(repoName, frequency string) {
	if gs.subscriptions[repoName] {
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

	// 添加到订阅列表
	gs.subscriptions[repoName] = true

	// 添加到调度器
	switch frequency {
	case "daily":
		if err := gs.scheduler.AddDailyJob(repoName); err != nil {
			fmt.Printf("Failed to add daily job: %v\n", err)
			delete(gs.subscriptions, repoName)
			return
		}
	case "weekly":
		if err := gs.scheduler.AddWeeklyJob(repoName); err != nil {
			fmt.Printf("Failed to add weekly job: %v\n", err)
			delete(gs.subscriptions, repoName)
			return
		}
	default:
		fmt.Printf("Invalid frequency: %s. Use 'daily' or 'weekly'\n", frequency)
		delete(gs.subscriptions, repoName)
		return
	}

	fmt.Printf("✓ Subscribed to %s (%s updates)\n", repoName, frequency)
}

func (gs *GitHubSentinel) unsubscribe(repoName string) {
	if !gs.subscriptions[repoName] {
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
	for repo := range gs.subscriptions {
		fmt.Printf("  • %s\n", repo)
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
	filename := fmt.Sprintf("%s_report_%s.md",
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
		filename := fmt.Sprintf("%s_report_%s.md",
			strings.ReplaceAll(repoName, "/", "_"),
			time.Now().Format("20060102_150405"))
		os.WriteFile(filename, []byte(report), 0644)
	}

	fmt.Println("\n✓ All reports generated")
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
		for repo := range gs.subscriptions {
			fmt.Printf("  • %s\n", repo)
		}
	}
}

func (gs *GitHubSentinel) clearScreen() {
	fmt.Print("\033[2J\033[H")
	gs.printHelp()
}

func (gs *GitHubSentinel) printHelp() {
	fmt.Println("\nGitHub Sentinel - Interactive Console")
	fmt.Println("=====================================")
	fmt.Println("Commands:")
	fmt.Println("  subscribe <owner/repo> [daily|weekly]  - Subscribe to a repository")
	fmt.Println("  unsubscribe <owner/repo>               - Unsubscribe from a repository")
	fmt.Println("  list, ls                               - List all subscriptions")
	fmt.Println("  fetch <owner/repo> [days]              - Fetch updates immediately")
	fmt.Println("  fetch-all [days]                       - Fetch all subscriptions")
	fmt.Println("  status, stat                           - Show system status")
	fmt.Println("  clear                                  - Clear screen")
	fmt.Println("  help, h, ?                             - Show this help")
	fmt.Println("  quit, exit, q                          - Exit program")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  subscribe langchain-ai/langchain daily")
	fmt.Println("  subscribe kubernetes/kubernetes weekly")
	fmt.Println("  fetch langchain-ai/langchain 7")
	fmt.Println("  fetch-all 3")
	fmt.Println("  list")
}

func parseRepoFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullName
}

func main() {
	// 获取 GitHub Token
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		fmt.Println("Warning: GITHUB_TOKEN not set, API rate limit will be very low")
		fmt.Println("You can set it with: export GITHUB_TOKEN=your_token_here")
		fmt.Println()
	}

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

// internal/scheduler/scheduler.go
package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SUNmingfeng/github-sentinel/internal/aggregator"
	"github.com/SUNmingfeng/github-sentinel/internal/service"
	"github.com/robfig/cron/v3"
)

type ReportHandler func(repoName string, report string, eventsCount int)

type Scheduler struct {
	cron           *cron.Cron
	repoService    *service.RepoService
	aggregator     *aggregator.Aggregator
	jobs           map[string]cron.EntryID
	mu             sync.RWMutex
	running        bool
	reportHandlers []ReportHandler
}

func NewScheduler(repoService *service.RepoService) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		repoService: repoService,
		aggregator:  aggregator.NewAggregator(),
		jobs:        make(map[string]cron.EntryID),
		running:     false,
	}
}

// 添加报告处理器
func (s *Scheduler) AddReportHandler(handler ReportHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reportHandlers = append(s.reportHandlers, handler)
}

func (s *Scheduler) AddDailyJob(repoName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已经存在
	if _, exists := s.jobs[repoName+"_daily"]; exists {
		return fmt.Errorf("daily job for %s already exists", repoName)
	}

	// 添加定时任务：每天早上9点执行
	entryID, err := s.cron.AddFunc("0 0 9 * * *", func() {
		s.runRepoCheck(repoName, 7)
	})
	if err != nil {
		return err
	}

	s.jobs[repoName+"_daily"] = entryID
	log.Printf("Added daily job for %s", repoName)
	return nil
}

func (s *Scheduler) AddWeeklyJob(repoName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已经存在
	if _, exists := s.jobs[repoName+"_weekly"]; exists {
		return fmt.Errorf("weekly job for %s already exists", repoName)
	}

	// 添加定时任务：每周一早上10点执行
	entryID, err := s.cron.AddFunc("0 0 10 * * 1", func() {
		s.runRepoCheck(repoName, 7)
	})
	if err != nil {
		return err
	}

	s.jobs[repoName+"_weekly"] = entryID
	log.Printf("Added weekly job for %s", repoName)
	return nil
}

func (s *Scheduler) RemoveJob(repoName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 移除 daily job
	if entryID, exists := s.jobs[repoName+"_daily"]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, repoName+"_daily")
		log.Printf("Removed daily job for %s", repoName)
	}

	// 移除 weekly job
	if entryID, exists := s.jobs[repoName+"_weekly"]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, repoName+"_weekly")
		log.Printf("Removed weekly job for %s", repoName)
	}
}

func (s *Scheduler) runRepoCheck(repoName string, days int) {
	log.Printf("Running scheduled check for %s", repoName)

	ctx := context.Background()
	events, release, err := s.repoService.FetchRepoUpdates(ctx, repoName, days)
	if err != nil {
		log.Printf("Error checking repo %s: %v", repoName, err)
		return
	}

	// 生成报告
	report := s.aggregator.GenerateReport(repoName, release, events)

	// 保存报告到文件
	filename := s.saveReport(repoName, report)
	log.Printf("Generated report for %s: %s (%d events)", repoName, filename, len(events))

	// 调用所有报告处理器（用于通知等）
	s.mu.RLock()
	handlers := s.reportHandlers
	s.mu.RUnlock()

	for _, handler := range handlers {
		handler(repoName, report, len(events))
	}
}

func (s *Scheduler) saveReport(repoName, report string) string {
	// 创建 reports 目录
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		log.Printf("Failed to create reports directory: %v", err)
		return ""
	}

	// 生成文件名
	safeName := strings.ReplaceAll(repoName, "/", "_")
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s/%s_%s.md", reportsDir, safeName, timestamp)

	// 保存报告
	if err := os.WriteFile(filename, []byte(report), 0644); err != nil {
		log.Printf("Failed to save report: %v", err)
		return ""
	}

	return filename
}

func (s *Scheduler) Start() {
	if s.running {
		return
	}
	s.cron.Start()
	s.running = true
	log.Println("Scheduler started")
}

func (s *Scheduler) Stop() {
	if !s.running {
		return
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	log.Println("Scheduler stopped")
}

func (s *Scheduler) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.running {
		return fmt.Sprintf("Running with %d active jobs", len(s.jobs))
	}
	return "Stopped"
}

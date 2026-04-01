// internal/service/repo_service.go
package service

import (
	"context"
	"fmt"
	"github.com/SUNmingfeng/github-sentinel/internal/domain"
	"github.com/SUNmingfeng/github-sentinel/internal/github"
	"strings"
)

type RepoService struct {
	githubClient *github.Client
}

func NewRepoService(githubClient *github.Client) *RepoService {
	return &RepoService{
		githubClient: githubClient,
	}
}

func (s *RepoService) FetchRepoUpdates(ctx context.Context, repoFullName string, days int) ([]domain.RepoEvent, *domain.Release, error) {
	owner, repo := parseRepoFullName(repoFullName)

	// 获取最新版本
	releaseInfo, err := s.githubClient.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		// 如果获取release失败，不中断流程，只是记录
		fmt.Printf("Warning: failed to get latest release: %v\n", err)
	}

	// 获取最近的提交
	commits, err := s.githubClient.GetRecentCommits(ctx, owner, repo, days)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get commits: %w", err)
	}

	// 获取最近的PR
	pulls, err := s.githubClient.GetRecentPullRequests(ctx, owner, repo, days)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	// 获取最近的Issues
	issues, err := s.githubClient.GetRecentIssues(ctx, owner, repo, days)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get issues: %w", err)
	}

	// 转换为通用事件
	var events []domain.RepoEvent

	for _, commit := range commits {
		// 截取commit消息的第一行
		message := commit.Message
		if idx := strings.Index(message, "\n"); idx > 0 {
			message = message[:idx]
		}

		events = append(events, domain.RepoEvent{
			Type:      "commit",
			Title:     fmt.Sprintf("%s: %s", commit.SHA[:7], message),
			Content:   commit.Message,
			URL:       commit.URL,
			Author:    commit.Author,
			EventDate: commit.Date,
		})
	}

	for _, pull := range pulls {
		events = append(events, domain.RepoEvent{
			Type:      "pull_request",
			Title:     fmt.Sprintf("#%d: %s", pull.Number, pull.Title),
			Content:   fmt.Sprintf("State: %s | Author: %s", pull.State, pull.Author),
			URL:       pull.URL,
			Author:    pull.Author,
			EventDate: pull.CreatedAt,
		})
	}

	for _, issue := range issues {
		events = append(events, domain.RepoEvent{
			Type:      "issue",
			Title:     fmt.Sprintf("#%d: %s", issue.Number, issue.Title),
			Content:   fmt.Sprintf("State: %s | Author: %s", issue.State, issue.Author),
			URL:       issue.URL,
			Author:    issue.Author,
			EventDate: issue.CreatedAt,
		})
	}

	var domainRelease *domain.Release
	if releaseInfo != nil {
		domainRelease = &domain.Release{
			TagName:     releaseInfo.TagName,
			Name:        releaseInfo.Name,
			Body:        releaseInfo.Body,
			URL:         releaseInfo.URL,
			PublishedAt: releaseInfo.PublishedAt,
		}
	}

	return events, domainRelease, nil
}

func parseRepoFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullName
}

// internal/github/client.go
package github

import (
	"context"
	"fmt"
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

type PullRequestInfo struct {
	Number    int
	Title     string
	URL       string
	Author    string
	State     string
	CreatedAt time.Time
}

type IssueInfo struct {
	Number    int
	Title     string
	URL       string
	Author    string
	State     string
	CreatedAt time.Time
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

func (c *Client) GetRecentPullRequests(ctx context.Context, owner, repo string, days int) ([]PullRequestInfo, error) {
	opts := &github.PullRequestListOptions{
		State:     "all",
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 10,
			Page:    1,
		},
	}

	pulls, resp, err := c.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}
	defer resp.Body.Close()

	var result []PullRequestInfo
	since := time.Now().AddDate(0, 0, -days)

	for _, pull := range pulls {
		if pull.GetCreatedAt().After(since) {
			result = append(result, PullRequestInfo{
				Number:    pull.GetNumber(),
				Title:     pull.GetTitle(),
				URL:       pull.GetHTMLURL(),
				Author:    pull.GetUser().GetLogin(),
				State:     pull.GetState(),
				CreatedAt: pull.GetCreatedAt().Time,
			})
		}
	}

	return result, nil
}

func (c *Client) GetRecentIssues(ctx context.Context, owner, repo string, days int) ([]IssueInfo, error) {
	opts := &github.IssueListByRepoOptions{
		State: "all",
		Sort:  "updated",
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}

	issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get issues: %w", err)
	}
	defer resp.Body.Close()

	var result []IssueInfo
	since := time.Now().AddDate(0, 0, -days)

	for _, issue := range issues {
		if issue.GetCreatedAt().After(since) && !issue.IsPullRequest() {
			result = append(result, IssueInfo{
				Number:    issue.GetNumber(),
				Title:     issue.GetTitle(),
				URL:       issue.GetHTMLURL(),
				Author:    issue.GetUser().GetLogin(),
				State:     issue.GetState(),
				CreatedAt: issue.GetCreatedAt().Time,
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

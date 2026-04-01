package github

import "context"

type Client interface {
GetCommits(ctx context.Context, owner, repo string) ([]Commit, error)
GetPullRequests(ctx context.Context, owner, repo string) ([]PullRequest, error)
GetIssues(ctx context.Context, owner, repo string) ([]Issue, error)
}

type Commit struct {
SHA     string
Message string
URL     string
}

type PullRequest struct {
Title string
URL   string
}

type Issue struct {
Title string
URL   string
}

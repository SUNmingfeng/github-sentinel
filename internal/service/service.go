package service

import (
"context"
"github.com/yourname/github-sentinel/internal/domain"
)

type SubscriptionService interface {
Subscribe(ctx context.Context, userId int64, repo string) error
Unsubscribe(ctx context.Context, userId int64, repo string) error
List(ctx context.Context, userId int64) ([]domain.Subscription, error)
}

type FetchService interface {
FetchRepoUpdates(ctx context.Context, repo domain.Repository) ([]domain.RepoEvent, error)
}

type ReportService interface {
GenerateReport(ctx context.Context, userId int64) (string, error)
}

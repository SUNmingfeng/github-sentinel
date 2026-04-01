package aggregator

import "github.com/yourname/github-sentinel/internal/domain"

type Aggregator struct{}

func (a *Aggregator) Aggregate(events []domain.RepoEvent) string {
result := "# Repo Updates\n\n"
for _, e := range events {
result += "- [" + e.Type + "] " + e.Title + "\n"
}
return result
}

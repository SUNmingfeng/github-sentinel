// internal/domain/models.go
package domain

import (
	"time"
)

type User struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Owner       string    `gorm:"index" json:"owner"`
	Name        string    `gorm:"index" json:"name"`
	FullName    string    `gorm:"uniqueIndex" json:"full_name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Stars       int       `json:"stars"`
	Forks       int       `json:"forks"`
	LastFetched time.Time `json:"last_fetched"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Subscription struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	UserID      int64     `gorm:"index" json:"user_id"`
	RepoID      int64     `gorm:"index" json:"repo_id"`
	Frequency   string    `json:"frequency"`    // daily, weekly
	NotifyTypes string    `json:"notify_types"` // commits, pulls, issues
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RepoEvent struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	RepoID    int64     `gorm:"index" json:"repo_id"`
	Type      string    `json:"type"` // commit, pull_request, issue, release
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	EventDate time.Time `json:"event_date"`
}

type Release struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	RepoID      int64     `gorm:"index" json:"repo_id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type Report struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"index" json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Format    string    `json:"format"` // markdown, html
	CreatedAt time.Time `json:"created_at"`
}

package domain

import "time"

type User struct {
Id    int64
Email string
}

type Repository struct {
Id       int64
Owner    string
Name     string
FullName string
}

type Subscription struct {
Id        int64
UserId    int64
RepoId    int64
Frequency string
CreatedAt time.Time
}

type RepoEvent struct {
Id        int64
RepoId    int64
Type      string
Title     string
Content   string
URL       string
CreatedAt time.Time
}

type Report struct {
Id        int64
UserId    int64
Content   string
CreatedAt time.Time
}

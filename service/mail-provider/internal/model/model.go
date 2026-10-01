package model

import "time"

const (
	DefaultMaxBodyBytes = 32 * 1024
	DefaultQueueSize    = 256
	DefaultWorkerCount  = 2
)

type Priority int32

const (
	PriorityUnspecified Priority = 0
	PriorityLow         Priority = 1
	PriorityMedium      Priority = 2
	PriorityHigh        Priority = 3
	PriorityUrgent      Priority = 4
)

type Subscription struct {
	ProfileID            string
	Email                string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt time.Time
	HistoryID            string
	WatchExpiresAt       time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SubscribeInput struct {
	ProfileID            string
	Email                string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt time.Time
}

type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type WatchResult struct {
	HistoryID string
	ExpiresAt time.Time
}

type Notification struct {
	ProfileID  string
	Email      string
	HistoryID  string
	MessageID  string
	ReceivedAt time.Time
}

type EmailMessage struct {
	ID         string
	ThreadID   string
	From       string
	To         string
	Subject    string
	Snippet    string
	Body       string
	ReceivedAt time.Time
}

type TaskDraft struct {
	Actionable  bool
	Title       string
	Description string
	Priority    Priority
	DueAt       *time.Time
}

type TaskInput struct {
	ProfileID   string
	Title       string
	Description string
	Priority    Priority
	DueAt       *time.Time
	Source      string
}

type TaskRef struct {
	ID    string
	Title string
}

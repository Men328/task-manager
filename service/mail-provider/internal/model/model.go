package model

import "time"

const (
	DefaultMaxBodyBytes       = 32 * 1024
	DefaultQueueSize          = 256
	DefaultWorkerCount        = 2
	DefaultMaxAttachments     = 5
	DefaultMaxAttachmentBytes = 10 << 20
)

type Priority int32

const (
	PriorityUnspecified Priority = 0
	PriorityLow         Priority = 1
	PriorityMedium      Priority = 2
	PriorityHigh        Priority = 3
	PriorityUrgent      Priority = 4
)

type Category string

const (
	CategoryTask     Category = "task"
	CategorySchedule Category = "schedule"
	CategoryEvent    Category = "event"
	CategoryOther    Category = "other"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryTask, CategorySchedule, CategoryEvent, CategoryOther:
		return true
	default:
		return false
	}
}

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

type EmailAttachment struct {
	Filename     string
	MimeType     string
	AttachmentID string
	Size         int64
	Data         []byte
}

type EmailMessage struct {
	ID          string
	ThreadID    string
	From        string
	To          string
	Subject     string
	Snippet     string
	Body        string
	ReceivedAt  time.Time
	Attachments []EmailAttachment
}

type MailDraft struct {
	Category    Category
	Actionable  bool
	Title       string
	Description string
	Priority    Priority
	DueAt       *time.Time
	StartAt     *time.Time
	EndAt       *time.Time
	AllDay      bool
	Location    string
	Reason      string
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

type ScheduleInput struct {
	ProfileID   string
	Title       string
	Description string
	Location    string
	StartAt     time.Time
	EndAt       *time.Time
	AllDay      bool
}

type ScheduleRef struct {
	ID    string
	Title string
}

type EventInput struct {
	ProfileID   string
	Title       string
	Description string
	Location    string
	StartAt     time.Time
	EndAt       *time.Time
	AllDay      bool
	Source      string
}

type EventRef struct {
	ID    string
	Title string
}

type BacklogInput struct {
	ProfileID   string
	Title       string
	Description string
	Sender      string
	Source      string
	Category    string
	Reason      string
	ObjectKey   string
}

type BacklogRef struct {
	ID    string
	Title string
}

const (
	NoticeTargetTask     = "task"
	NoticeTargetSchedule = "schedule"
	NoticeTargetEvent    = "event"
	NoticeTargetBacklog  = "backlog"
)

const (
	NoticeTypeTaskCreated     = "TASK_CREATED"
	NoticeTypeScheduleCreated = "SCHEDULE_CREATED"
	NoticeTypeEventCreated    = "EVENT_CREATED"
	NoticeTypeBacklogCreated  = "BACKLOG_CREATED"
)

type NoticeInput struct {
	ProfileID  string
	Type       string
	Title      string
	Body       string
	TargetType string
	TargetID   string
	Source     string
}

type NoticeRef struct {
	ID    string
	Title string
}

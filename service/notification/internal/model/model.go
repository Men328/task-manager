package model

import "time"

const (
	TargetTypeTask     = "task"
	TargetTypeSchedule = "schedule"
	TargetTypeEvent    = "event"
	TargetTypeBacklog  = "backlog"
)

const (
	TypeTaskCreated     = "TASK_CREATED"
	TypeScheduleCreated = "SCHEDULE_CREATED"
	TypeEventCreated    = "EVENT_CREATED"
	TypeBacklogCreated  = "BACKLOG_CREATED"
)

const (
	MarkAllID     = "*"
	DefaultLimit  = 20
	MaxLimit      = 100
	DefaultPrefix = "noti-internal-"
	PrivatePrefix = "private-"
)

type Notice struct {
	ID         string
	ProfileID  string
	Type       string
	Title      string
	Body       string
	TargetType string
	TargetID   string
	Source     string
	IsRead     bool
	CreatedAt  time.Time
	ReadAt     *time.Time
}

type Filter struct {
	ProfileID  string
	Limit      int
	UnreadOnly bool
}

func TargetTypeValid(targetType string) bool {
	switch targetType {
	case TargetTypeTask, TargetTypeSchedule, TargetTypeEvent, TargetTypeBacklog:
		return true
	default:
		return false
	}
}

func ChannelName(prefix string, profileID string) string {
	return PrivatePrefix + prefix + profileID
}

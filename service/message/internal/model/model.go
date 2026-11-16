package model

import "time"

const (
	MaxBodyRunes            = 2000
	MaxClientIDBytes        = 64
	MaxTaskTitleRunes       = 200
	MaxTaskDescriptionRunes = 4000
)

type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	RecipientID    string
	Body           string
	ClientID       string
	CreatedAt      time.Time
	Task           *TaskRef
}

type TaskRef struct {
	TaskID          string
	Title           string
	StatusName      string
	StatusCategory  string
	StatusColor     string
	StatusChangedAt time.Time
}

type TaskDraft struct {
	Title       string
	Description string
	Priority    string
	DueAt       *time.Time
}

type TaskRequest struct {
	RecipientID string
	RequestedBy string
	MessageID   string
	Draft       TaskDraft
}

type Conversation struct {
	ID          string
	PeerID      string
	LastMessage Message
}

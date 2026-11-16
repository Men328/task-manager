package model

import (
	"errors"
	"time"
)

var ErrPermanent = errors.New("sự kiện không thể xử lý")

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

type TaskStatusChanged struct {
	EventID        string
	TaskID         string
	OwnerID        string
	Title          string
	RequestedBy    string
	MessageID      string
	FromStatusName string
	ToStatusName   string
	ToCategory     string
	ToColor        string
	ChangedAt      time.Time
}

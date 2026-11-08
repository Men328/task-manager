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
}

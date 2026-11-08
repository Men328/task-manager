package model

import "time"

const (
	MaxBodyRunes     = 2000
	MaxClientIDBytes = 64
)

type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	RecipientID    string
	Body           string
	ClientID       string
	CreatedAt      time.Time
}

type Conversation struct {
	ID          string
	PeerID      string
	LastMessage Message
}

package service

import (
	"context"
	"time"

	"taskmanager/service/message/internal/model"
)

type MessageReader interface {
	GetMessage(ctx context.Context, id string) (model.Message, error)
	ListMessages(ctx context.Context, conversationID string, before time.Time, limit int) ([]model.Message, error)
	ListConversations(ctx context.Context, profileID string, limit int) ([]model.Conversation, error)
}

type FriendChecker interface {
	AreFriends(ctx context.Context, profileID, otherID string) (bool, error)
}

type TaskCreator interface {
	CreateForMessage(ctx context.Context, request model.TaskRequest) (model.TaskRef, error)
}

type Broadcaster interface {
	Broadcast(ctx context.Context, message model.Message) error
}

type EventPublisher interface {
	Publish(ctx context.Context, message model.Message) error
}

type MessageService interface {
	Send(ctx context.Context, profileID, peerID, body, clientID string) (model.Message, error)
	SendTask(ctx context.Context, profileID, peerID string, draft model.TaskDraft, clientID string) (model.Message, error)
	Get(ctx context.Context, profileID, messageID string) (model.Message, error)
	ListMessages(ctx context.Context, profileID, peerID string, before time.Time, limit int) ([]model.Message, bool, error)
	ListConversations(ctx context.Context, profileID string) ([]model.Conversation, error)
}

package service

import (
	"context"

	"taskmanager/service/worker/internal/model"
)

type MessageStore interface {
	Save(ctx context.Context, message model.Message) error
	UpdateTaskStatus(ctx context.Context, event model.TaskStatusChanged) (model.Message, bool, error)
}

type MessageBroadcaster interface {
	BroadcastUpdated(ctx context.Context, message model.Message) error
}

type TaskNotifier interface {
	TaskStatusChanged(ctx context.Context, event model.TaskStatusChanged) error
}

type TaskEventHandler interface {
	Handle(ctx context.Context, event model.TaskStatusChanged) error
}

type MessagePersister interface {
	Persist(ctx context.Context, message model.Message) error
}

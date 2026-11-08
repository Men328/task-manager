package service

import (
	"context"

	"taskmanager/service/worker/internal/model"
)

type MessageStore interface {
	Save(ctx context.Context, message model.Message) error
}

type MessagePersister interface {
	Persist(ctx context.Context, message model.Message) error
}

package handler

import (
	"context"

	"taskmanager/service/worker/internal/dependency"
	"taskmanager/service/worker/internal/service"
)

type ChatMessageProcessor struct {
	persister service.MessagePersister
}

func NewChatMessageProcessor(persister service.MessagePersister) *ChatMessageProcessor {
	return &ChatMessageProcessor{persister: persister}
}

func (p *ChatMessageProcessor) Process(ctx context.Context, value []byte) error {
	message, err := dependency.DecodeMessageEvent(value)
	if err != nil {
		return err
	}
	return p.persister.Persist(ctx, message)
}

type TaskEventProcessor struct {
	handler service.TaskEventHandler
}

func NewTaskEventProcessor(handler service.TaskEventHandler) *TaskEventProcessor {
	return &TaskEventProcessor{handler: handler}
}

func (p *TaskEventProcessor) Process(ctx context.Context, value []byte) error {
	event, err := dependency.DecodeTaskStatusChanged(value)
	if err != nil {
		return err
	}
	return p.handler.Handle(ctx, event)
}

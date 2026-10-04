package repository

import (
	"context"

	"taskmanager/service/notification/internal/model"
)

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (p *NoopPublisher) Publish(_ context.Context, _ model.Notice) error {
	return nil
}

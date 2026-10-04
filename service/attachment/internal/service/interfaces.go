package service

import (
	"context"

	"taskmanager/service/attachment/internal/model"
)

type AttachmentRepository interface {
	Create(ctx context.Context, item model.Attachment) (model.Attachment, error)
	Get(ctx context.Context, id string) (model.Attachment, error)
	List(ctx context.Context, f model.Filter) ([]model.Attachment, error)
	Delete(ctx context.Context, id string) error
}

type BlobStore interface {
	Put(ctx context.Context, key string, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Remove(ctx context.Context, key string) error
}

type AttachmentService interface {
	Upload(ctx context.Context, in model.Upload) (model.Attachment, error)
	Get(ctx context.Context, id string) (model.Attachment, error)
	Download(ctx context.Context, id string) (model.Attachment, []byte, error)
	List(ctx context.Context, f model.Filter) ([]model.Attachment, error)
	Delete(ctx context.Context, id string) error
}

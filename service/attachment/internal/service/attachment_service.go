package service

import (
	"context"
	"path"
	"strings"

	"github.com/google/uuid"

	"taskmanager/service/attachment/internal/model"
)

const objectKeyPrefix = "attachments"

type attachmentService struct {
	items    AttachmentRepository
	blobs    BlobStore
	maxBytes int64
}

func NewAttachmentService(items AttachmentRepository, blobs BlobStore, maxBytes int64) AttachmentService {
	if maxBytes <= 0 {
		maxBytes = 10 << 20
	}
	return &attachmentService{items: items, blobs: blobs, maxBytes: maxBytes}
}

func (s *attachmentService) Upload(ctx context.Context, in model.Upload) (model.Attachment, error) {
	ownerType, ok := model.NormalizeOwnerType(in.OwnerType)
	if !ok {
		return model.Attachment{}, model.NewError(
			model.ErrorKindOwnerTypeInvalid, "unsupported owner type %q", in.OwnerType,
		)
	}

	fileName := strings.TrimSpace(in.FileName)
	if fileName == "" {
		return model.Attachment{}, model.NewError(model.ErrorKindFileRequired, "file name is required")
	}

	contentType := in.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	size := int64(len(in.Content))
	if size == 0 {
		return model.Attachment{}, model.NewError(model.ErrorKindFileRequired, "file content is empty")
	}
	if size > s.maxBytes {
		return model.Attachment{}, model.NewError(
			model.ErrorKindFileTooLarge, "attachment exceeds %d bytes", s.maxBytes,
		)
	}

	item := model.Attachment{
		ProfileID:   in.ProfileID,
		OwnerType:   ownerType,
		OwnerID:     in.OwnerID,
		FileName:    fileName,
		ContentType: contentType,
		Size:        size,
		ObjectKey:   buildObjectKey(in.ProfileID, ownerType, in.OwnerID, fileName),
	}

	if err := s.blobs.Put(ctx, item.ObjectKey, contentType, in.Content); err != nil {
		return model.Attachment{}, err
	}

	created, err := s.items.Create(ctx, item)
	if err != nil {
		_ = s.blobs.Remove(ctx, item.ObjectKey)
		return model.Attachment{}, err
	}
	return created, nil
}

func (s *attachmentService) Get(ctx context.Context, id string) (model.Attachment, error) {
	return s.items.Get(ctx, id)
}

func (s *attachmentService) Download(ctx context.Context, id string) (model.Attachment, []byte, error) {
	item, err := s.items.Get(ctx, id)
	if err != nil {
		return model.Attachment{}, nil, err
	}
	data, err := s.blobs.Get(ctx, item.ObjectKey)
	if err != nil {
		return model.Attachment{}, nil, err
	}
	return item, data, nil
}

func (s *attachmentService) List(ctx context.Context, f model.Filter) ([]model.Attachment, error) {
	if f.OwnerType != "" {
		ownerType, ok := model.NormalizeOwnerType(f.OwnerType)
		if !ok {
			return nil, model.NewError(
				model.ErrorKindOwnerTypeInvalid, "unsupported owner type %q", f.OwnerType,
			)
		}
		f.OwnerType = ownerType
	}
	return s.items.List(ctx, f)
}

func (s *attachmentService) Delete(ctx context.Context, id string) error {
	item, err := s.items.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.items.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.blobs.Remove(ctx, item.ObjectKey)
	return nil
}

func buildObjectKey(profileID string, ownerType string, ownerID string, fileName string) string {
	return path.Join(objectKeyPrefix, profileID, ownerType, ownerID, uuid.NewString()+safeExt(fileName))
}

func safeExt(fileName string) string {
	ext := strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
	if len(ext) < 2 || len(ext) > 16 {
		return ""
	}
	for _, r := range ext[1:] {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return ""
	}
	return ext
}

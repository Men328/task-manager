package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"taskmanager/service/attachment/internal/model"
)

type fakeRepository struct {
	created []model.Attachment
	items   map[string]model.Attachment
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: make(map[string]model.Attachment)}
}

func (r *fakeRepository) Create(_ context.Context, item model.Attachment) (model.Attachment, error) {
	item.ID = fmt.Sprintf("att-%d", len(r.items)+1)
	r.items[item.ID] = item
	r.created = append(r.created, item)
	return item, nil
}

func (r *fakeRepository) Get(_ context.Context, id string) (model.Attachment, error) {
	item, ok := r.items[id]
	if !ok {
		return model.Attachment{}, model.ErrNotFound
	}
	return item, nil
}

func (r *fakeRepository) List(_ context.Context, f model.Filter) ([]model.Attachment, error) {
	out := make([]model.Attachment, 0)
	for _, item := range r.items {
		if f.OwnerType != "" && item.OwnerType != f.OwnerType {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

type fakeBlobStore struct {
	objects map[string][]byte
	putErr  error
}

func newFakeBlobStore() *fakeBlobStore {
	return &fakeBlobStore{objects: make(map[string][]byte)}
}

func (s *fakeBlobStore) Put(_ context.Context, key string, _ string, data []byte) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.objects[key] = append([]byte(nil), data...)
	return nil
}

func (s *fakeBlobStore) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return data, nil
}

func (s *fakeBlobStore) Remove(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func TestUploadValidatesOwnerType(t *testing.T) {
	svc := NewAttachmentService(newFakeRepository(), newFakeBlobStore(), 1024)

	_, err := svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: "workspace",
		OwnerID:   "o1",
		FileName:  "a.txt",
		Content:   []byte("hello"),
	})

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindOwnerTypeInvalid {
		t.Fatalf("expected owner_type_invalid, got %v", err)
	}
}

func TestUploadRejectsEmptyAndOversizedFiles(t *testing.T) {
	svc := NewAttachmentService(newFakeRepository(), newFakeBlobStore(), 4)

	_, err := svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: model.OwnerTypeTask,
		OwnerID:   "o1",
		FileName:  "a.txt",
	})
	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindFileRequired {
		t.Fatalf("expected file_required, got %v", err)
	}

	_, err = svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: model.OwnerTypeTask,
		OwnerID:   "o1",
		FileName:  "a.txt",
		Content:   []byte("hello"),
	})
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindFileTooLarge {
		t.Fatalf("expected file_too_large, got %v", err)
	}
}

func TestUploadStoresBlobAndMetadata(t *testing.T) {
	repo := newFakeRepository()
	blobs := newFakeBlobStore()
	svc := NewAttachmentService(repo, blobs, 1024)

	created, err := svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: "TASK",
		OwnerID:   "o1",
		FileName:  "  report.pdf  ",
		Content:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.OwnerType != model.OwnerTypeTask {
		t.Fatalf("owner type = %q, want task", created.OwnerType)
	}
	if created.FileName != "report.pdf" {
		t.Fatalf("file name = %q, want trimmed", created.FileName)
	}
	if created.ContentType != "application/octet-stream" {
		t.Fatalf("content type = %q, want default", created.ContentType)
	}
	if created.Size != 5 {
		t.Fatalf("size = %d, want 5", created.Size)
	}
	if created.ObjectKey == "" {
		t.Fatal("object key is empty")
	}
	if string(blobs.objects[created.ObjectKey]) != "hello" {
		t.Fatalf("blob content = %q, want hello", string(blobs.objects[created.ObjectKey]))
	}
	if len(repo.created) != 1 {
		t.Fatalf("repository create calls = %d, want 1", len(repo.created))
	}
}

func TestUploadReturnsStorageError(t *testing.T) {
	repo := newFakeRepository()
	blobs := newFakeBlobStore()
	blobs.putErr = fmt.Errorf("%w: boom", model.ErrStorageFailed)
	svc := NewAttachmentService(repo, blobs, 1024)

	_, err := svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: model.OwnerTypeTask,
		OwnerID:   "o1",
		FileName:  "a.txt",
		Content:   []byte("hello"),
	})
	if !errors.Is(err, model.ErrStorageFailed) {
		t.Fatalf("expected storage failure, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("repository create calls = %d, want 0", len(repo.created))
	}
}

func TestDownloadReadsBlob(t *testing.T) {
	svc := NewAttachmentService(newFakeRepository(), newFakeBlobStore(), 1024)

	created, err := svc.Upload(context.Background(), model.Upload{
		ProfileID: "p1",
		OwnerType: model.OwnerTypeTask,
		OwnerID:   "o1",
		FileName:  "a.txt",
		Content:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item, data, err := svc.Download(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected download error: %v", err)
	}
	if item.ID != created.ID {
		t.Fatalf("downloaded id = %q, want %q", item.ID, created.ID)
	}
	if string(data) != "hello" {
		t.Fatalf("downloaded content = %q, want hello", string(data))
	}
}

func TestListRejectsUnknownOwnerType(t *testing.T) {
	svc := NewAttachmentService(newFakeRepository(), newFakeBlobStore(), 1024)

	_, err := svc.List(context.Background(), model.Filter{OwnerType: "note"})
	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindOwnerTypeInvalid {
		t.Fatalf("expected owner_type_invalid, got %v", err)
	}
}

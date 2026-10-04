package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/attachment/internal/model"
)

type InMemoryAttachmentRepository struct {
	mu    sync.RWMutex
	items map[string]model.Attachment
}

func NewInMemoryAttachmentRepository() *InMemoryAttachmentRepository {
	return &InMemoryAttachmentRepository{items: make(map[string]model.Attachment)}
}

func (r *InMemoryAttachmentRepository) Create(_ context.Context, item model.Attachment) (model.Attachment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	r.items[item.ID] = item
	return item, nil
}

func (r *InMemoryAttachmentRepository) Get(_ context.Context, id string) (model.Attachment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return model.Attachment{}, fmt.Errorf("%w: attachment %s", model.ErrNotFound, id)
	}
	return item, nil
}

func (r *InMemoryAttachmentRepository) List(_ context.Context, f model.Filter) ([]model.Attachment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Attachment, 0, len(r.items))
	for _, item := range r.items {
		if f.ProfileID != "" && item.ProfileID != f.ProfileID {
			continue
		}
		if f.OwnerType != "" && item.OwnerType != f.OwnerType {
			continue
		}
		if f.OwnerID != "" && item.OwnerID != f.OwnerID {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func (r *InMemoryAttachmentRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: attachment %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

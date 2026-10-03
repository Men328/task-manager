package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/backlog/internal/model"
)

type InMemoryBacklogRepository struct {
	mu    sync.RWMutex
	items map[string]model.Item
}

func NewInMemoryBacklogRepository() *InMemoryBacklogRepository {
	return &InMemoryBacklogRepository{items: make(map[string]model.Item)}
}

func (r *InMemoryBacklogRepository) Create(_ context.Context, item model.Item) (model.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.Status == "" {
		item.Status = model.StatusNew
	}
	if item.Category == "" {
		item.Category = model.CategoryOther
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	r.items[item.ID] = item
	return item, nil
}

func (r *InMemoryBacklogRepository) Get(_ context.Context, id string) (model.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return model.Item{}, fmt.Errorf("%w: backlog %s", model.ErrNotFound, id)
	}
	return item, nil
}

func (r *InMemoryBacklogRepository) List(_ context.Context, f model.Filter) ([]model.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Item, 0, len(r.items))
	for _, item := range r.items {
		if f.ProfileID != "" && item.ProfileID != f.ProfileID {
			continue
		}
		if f.Status != "" && item.Status != f.Status {
			continue
		}
		if f.Category != "" && item.Category != f.Category {
			continue
		}
		out = append(out, item)
	}
	sortItems(out)
	return out, nil
}

func (r *InMemoryBacklogRepository) Update(_ context.Context, id string, upd model.Update) (model.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return model.Item{}, fmt.Errorf("%w: backlog %s", model.ErrNotFound, id)
	}
	if upd.Title != nil {
		item.Title = *upd.Title
	}
	if upd.Description != nil {
		item.Description = *upd.Description
	}
	if upd.Reason != nil {
		item.Reason = *upd.Reason
	}
	if upd.Status != nil {
		item.Status = *upd.Status
	}
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
	return item, nil
}

func (r *InMemoryBacklogRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: backlog %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

func sortItems(items []model.Item) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID > items[j].ID
	})
}

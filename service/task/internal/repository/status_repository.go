package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/task/internal/model"
)

type InMemoryStatusRepository struct {
	mu    sync.RWMutex
	items map[string]model.Status
}

func NewInMemoryStatusRepository() *InMemoryStatusRepository {
	return &InMemoryStatusRepository{items: make(map[string]model.Status)}
}

func (r *InMemoryStatusRepository) Create(_ context.Context, s model.Status) (model.Status, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	if s.IsDefault {
		for id, existing := range r.items {
			if existing.ProfileID == s.ProfileID && existing.IsDefault && !existing.IsArchived {
				existing.IsDefault = false
				existing.UpdatedAt = now
				r.items[id] = existing
			}
		}
	}
	r.items[s.ID] = s
	return s, nil
}

func (r *InMemoryStatusRepository) Get(_ context.Context, id string) (model.Status, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.items[id]
	if !ok {
		return model.Status{}, fmt.Errorf("%w: status %s", model.ErrNotFound, id)
	}
	return s, nil
}

func (r *InMemoryStatusRepository) List(_ context.Context, profileID string, includeArchived bool) ([]model.Status, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Status, 0, len(r.items))
	for _, s := range r.items {
		if profileID != "" && s.ProfileID != profileID {
			continue
		}
		if !includeArchived && s.IsArchived {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (r *InMemoryStatusRepository) Update(_ context.Context, id string, upd model.StatusUpdate) (model.Status, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.items[id]
	if !ok {
		return model.Status{}, fmt.Errorf("%w: status %s", model.ErrNotFound, id)
	}
	now := time.Now().UTC()
	if upd.Name != nil {
		s.Name = *upd.Name
	}
	if upd.Description != nil {
		s.Description = *upd.Description
	}
	if upd.Color != nil {
		s.Color = *upd.Color
	}
	if upd.Category != nil {
		s.Category = *upd.Category
	}
	if upd.IsTerminal != nil {
		s.IsTerminal = *upd.IsTerminal
	}
	if upd.Position != nil {
		s.Position = *upd.Position
	}
	if upd.IsArchived != nil {
		s.IsArchived = *upd.IsArchived
	}
	if upd.IsDefault != nil && *upd.IsDefault {
		for otherID, existing := range r.items {
			if otherID != id && existing.ProfileID == s.ProfileID && existing.IsDefault {
				existing.IsDefault = false
				existing.UpdatedAt = now
				r.items[otherID] = existing
			}
		}
		s.IsDefault = true
	} else if upd.IsDefault != nil {
		s.IsDefault = false
	}
	s.UpdatedAt = now
	r.items[id] = s
	return s, nil
}

func (r *InMemoryStatusRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: status %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

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

type InMemoryTransitionRepository struct {
	mu    sync.RWMutex
	items map[string]model.Transition
}

func NewInMemoryTransitionRepository() *InMemoryTransitionRepository {
	return &InMemoryTransitionRepository{items: make(map[string]model.Transition)}
}

func (r *InMemoryTransitionRepository) Create(_ context.Context, t model.Transition) (model.Transition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t.FromStatusID == t.ToStatusID {
		return model.Transition{}, fmt.Errorf("from_status_id và to_status_id không được trùng nhau")
	}
	for _, existing := range r.items {
		if existing.ProfileID == t.ProfileID &&
			existing.FromStatusID == t.FromStatusID &&
			existing.ToStatusID == t.ToStatusID {
			return model.Transition{}, fmt.Errorf("%w: rule %s -> %s", model.ErrAlreadyExists, t.FromStatusID, t.ToStatusID)
		}
	}

	now := time.Now().UTC()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	r.items[t.ID] = t
	return t, nil
}

func (r *InMemoryTransitionRepository) Get(_ context.Context, id string) (model.Transition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.items[id]
	if !ok {
		return model.Transition{}, fmt.Errorf("%w: transition %s", model.ErrNotFound, id)
	}
	return t, nil
}

func (r *InMemoryTransitionRepository) List(_ context.Context, profileID, fromStatusID string) ([]model.Transition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Transition, 0, len(r.items))
	for _, t := range r.items {
		if profileID != "" && t.ProfileID != profileID {
			continue
		}
		if fromStatusID != "" && t.FromStatusID != fromStatusID {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *InMemoryTransitionRepository) Update(_ context.Context, id string, upd model.TransitionUpdate) (model.Transition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.items[id]
	if !ok {
		return model.Transition{}, fmt.Errorf("%w: transition %s", model.ErrNotFound, id)
	}
	if upd.IsActive != nil {
		t.IsActive = *upd.IsActive
	}
	if upd.RequiresNote != nil {
		t.RequiresNote = *upd.RequiresNote
	}
	if upd.Description != nil {
		t.Description = *upd.Description
	}
	t.UpdatedAt = time.Now().UTC()
	r.items[id] = t
	return t, nil
}

func (r *InMemoryTransitionRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: transition %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

func (r *InMemoryTransitionRepository) IsAllowed(_ context.Context, profileID, fromStatusID, toStatusID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.items {
		if t.ProfileID == profileID && t.FromStatusID == fromStatusID && t.ToStatusID == toStatusID {
			return t.IsActive, nil
		}
	}
	return false, nil
}

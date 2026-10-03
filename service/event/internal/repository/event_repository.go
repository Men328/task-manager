package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/event/internal/model"
)

type InMemoryEventRepository struct {
	mu    sync.RWMutex
	items map[string]model.Event
}

func NewInMemoryEventRepository() *InMemoryEventRepository {
	return &InMemoryEventRepository{items: make(map[string]model.Event)}
}

func (r *InMemoryEventRepository) Create(_ context.Context, e model.Event) (model.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Status == "" {
		e.Status = model.StatusPlanned
	}
	e.CreatedAt = now
	e.UpdatedAt = now
	r.items[e.ID] = e
	return e, nil
}

func (r *InMemoryEventRepository) Get(_ context.Context, id string) (model.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.items[id]
	if !ok {
		return model.Event{}, fmt.Errorf("%w: event %s", model.ErrNotFound, id)
	}
	return e, nil
}

func (r *InMemoryEventRepository) List(_ context.Context, f model.Filter) ([]model.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Event, 0, len(r.items))
	for _, e := range r.items {
		if f.ProfileID != "" && e.ProfileID != f.ProfileID {
			continue
		}
		if f.Status != "" && e.Status != f.Status {
			continue
		}
		if !overlaps(e, f.From, f.To) {
			continue
		}
		out = append(out, e)
	}
	sortEvents(out)
	return out, nil
}

func (r *InMemoryEventRepository) Update(_ context.Context, id string, upd model.Update) (model.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.items[id]
	if !ok {
		return model.Event{}, fmt.Errorf("%w: event %s", model.ErrNotFound, id)
	}
	if upd.Title != nil {
		e.Title = *upd.Title
	}
	if upd.Description != nil {
		e.Description = *upd.Description
	}
	if upd.Location != nil {
		e.Location = *upd.Location
	}
	if upd.StartAt != nil {
		e.StartAt = *upd.StartAt
	}
	if upd.EndAt != nil {
		e.EndAt = upd.EndAt
	}
	if upd.AllDay != nil {
		e.AllDay = *upd.AllDay
	}
	if upd.Color != nil {
		e.Color = *upd.Color
	}
	if upd.Status != nil {
		e.Status = *upd.Status
	}
	e.UpdatedAt = time.Now().UTC()
	r.items[id] = e
	return e, nil
}

func (r *InMemoryEventRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: event %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

func overlaps(e model.Event, from, to *time.Time) bool {
	end := e.StartAt
	if e.EndAt != nil {
		end = *e.EndAt
	}
	if from != nil && end.Before(*from) {
		return false
	}
	if to != nil && !e.StartAt.Before(*to) {
		return false
	}
	return true
}

func sortEvents(items []model.Event) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].StartAt.Equal(items[j].StartAt) {
			return items[i].StartAt.Before(items[j].StartAt)
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}

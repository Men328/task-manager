package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/calendar/internal/model"
)

type InMemoryScheduleRepository struct {
	mu    sync.RWMutex
	items map[string]model.Schedule
}

func NewInMemoryScheduleRepository() *InMemoryScheduleRepository {
	return &InMemoryScheduleRepository{items: make(map[string]model.Schedule)}
}

func (r *InMemoryScheduleRepository) Create(_ context.Context, s model.Schedule) (model.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.CreatedAt = now
	s.UpdatedAt = now
	r.items[s.ID] = s
	return s, nil
}

func (r *InMemoryScheduleRepository) Get(_ context.Context, id string) (model.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.items[id]
	if !ok {
		return model.Schedule{}, fmt.Errorf("%w: schedule %s", model.ErrNotFound, id)
	}
	return s, nil
}

func (r *InMemoryScheduleRepository) List(_ context.Context, f model.ScheduleFilter) ([]model.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Schedule, 0, len(r.items))
	for _, s := range r.items {
		if f.ProfileID != "" && s.ProfileID != f.ProfileID {
			continue
		}
		if !overlaps(s, f.From, f.To) {
			continue
		}
		out = append(out, s)
	}
	sortSchedules(out)
	return out, nil
}

func (r *InMemoryScheduleRepository) Update(_ context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.items[id]
	if !ok {
		return model.Schedule{}, fmt.Errorf("%w: schedule %s", model.ErrNotFound, id)
	}
	if upd.Title != nil {
		s.Title = *upd.Title
	}
	if upd.Description != nil {
		s.Description = *upd.Description
	}
	if upd.Location != nil {
		s.Location = *upd.Location
	}
	if upd.StartAt != nil {
		s.StartAt = *upd.StartAt
	}
	if upd.EndAt != nil {
		s.EndAt = upd.EndAt
	}
	if upd.AllDay != nil {
		s.AllDay = *upd.AllDay
	}
	if upd.Color != nil {
		s.Color = *upd.Color
	}
	s.UpdatedAt = time.Now().UTC()
	r.items[id] = s
	return s, nil
}

func (r *InMemoryScheduleRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: schedule %s", model.ErrNotFound, id)
	}
	delete(r.items, id)
	return nil
}

func overlaps(s model.Schedule, from, to *time.Time) bool {
	end := s.StartAt
	if s.EndAt != nil {
		end = *s.EndAt
	}
	if from != nil && end.Before(*from) {
		return false
	}
	if to != nil && !s.StartAt.Before(*to) {
		return false
	}
	return true
}

func sortSchedules(items []model.Schedule) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].StartAt.Equal(items[j].StartAt) {
			return items[i].StartAt.Before(items[j].StartAt)
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}

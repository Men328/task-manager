package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/identity/internal/model"
)

type InMemoryProfileRepository struct {
	mu    sync.RWMutex
	items map[string]model.Profile
}

func NewInMemoryProfileRepository() *InMemoryProfileRepository {
	return &InMemoryProfileRepository{items: make(map[string]model.Profile)}
}

func (r *InMemoryProfileRepository) Create(_ context.Context, p model.Profile) (model.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.CreatedAt = now
	p.UpdatedAt = now
	r.items[p.ID] = p
	return p, nil
}

func (r *InMemoryProfileRepository) Get(_ context.Context, id string) (model.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.items[id]
	if !ok {
		return model.Profile{}, model.ErrNotFound
	}
	return p, nil
}

func (r *InMemoryProfileRepository) List(_ context.Context, limit int) ([]model.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Profile, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *InMemoryProfileRepository) Update(_ context.Context, id string, upd model.ProfileUpdate) (model.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.items[id]
	if !ok {
		return model.Profile{}, model.ErrNotFound
	}
	if upd.DisplayName != nil {
		p.DisplayName = *upd.DisplayName
	}
	if upd.AvatarURL != nil {
		p.AvatarURL = *upd.AvatarURL
	}
	if upd.Timezone != nil {
		p.Timezone = *upd.Timezone
	}
	if upd.Locale != nil {
		p.Locale = *upd.Locale
	}
	if upd.IsActive != nil {
		p.IsActive = *upd.IsActive
	}
	p.UpdatedAt = time.Now().UTC()
	r.items[id] = p
	return p, nil
}

func (r *InMemoryProfileRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/project/internal/model"
)

type InMemoryProjectRepository struct {
	mu    sync.RWMutex
	items map[string]model.Project
}

func NewInMemoryProjectRepository() *InMemoryProjectRepository {
	return &InMemoryProjectRepository{items: make(map[string]model.Project)}
}

func (r *InMemoryProjectRepository) nameTaken(profileID, name, exceptID string) bool {
	for _, it := range r.items {
		if it.ProfileID == profileID && it.ID != exceptID && strings.EqualFold(it.Name, name) {
			return true
		}
	}
	return false
}

func (r *InMemoryProjectRepository) Create(_ context.Context, p model.Project) (model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nameTaken(p.ProfileID, p.Name, "") {
		return model.Project{}, model.NewError(model.ErrorKindNameDuplicate, "project name already exists")
	}
	if p.IsDefault {
		for _, it := range r.items {
			if it.ProfileID == p.ProfileID && it.IsDefault {
				return model.Project{}, model.NewError(model.ErrorKindNameDuplicate, "default project already exists")
			}
		}
	}
	now := time.Now().UTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.CreatedAt = now
	p.UpdatedAt = now
	r.items[p.ID] = p
	return p, nil
}

func (r *InMemoryProjectRepository) Get(_ context.Context, id string) (model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.items[id]
	if !ok {
		return model.Project{}, fmt.Errorf("%w: project %s", model.ErrNotFound, id)
	}
	return p, nil
}

func (r *InMemoryProjectRepository) List(_ context.Context, f model.ProjectFilter) ([]model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Project, 0, len(r.items))
	for _, p := range r.items {
		if f.ProfileID != "" && p.ProfileID != f.ProfileID {
			continue
		}
		if p.IsArchived && !f.IncludeArchived {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (r *InMemoryProjectRepository) GetDefault(_ context.Context, profileID string) (model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.items {
		if p.ProfileID == profileID && p.IsDefault {
			return p, nil
		}
	}
	return model.Project{}, fmt.Errorf("%w: default project of %s", model.ErrNotFound, profileID)
}

func (r *InMemoryProjectRepository) Update(_ context.Context, id string, upd model.ProjectUpdate) (model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.items[id]
	if !ok {
		return model.Project{}, fmt.Errorf("%w: project %s", model.ErrNotFound, id)
	}
	if upd.Name != nil {
		if r.nameTaken(p.ProfileID, *upd.Name, id) {
			return model.Project{}, model.NewError(model.ErrorKindNameDuplicate, "project name already exists")
		}
		p.Name = *upd.Name
	}
	if upd.Description != nil {
		p.Description = *upd.Description
	}
	if upd.Color != nil {
		p.Color = *upd.Color
	}
	p.UpdatedAt = time.Now().UTC()
	r.items[id] = p
	return p, nil
}

func (r *InMemoryProjectRepository) SetArchived(_ context.Context, id string, archived bool) (model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.items[id]
	if !ok {
		return model.Project{}, fmt.Errorf("%w: project %s", model.ErrNotFound, id)
	}
	p.IsArchived = archived
	p.UpdatedAt = time.Now().UTC()
	r.items[id] = p
	return p, nil
}

package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/workspace/internal/model"
)

type InMemoryWorkspaceRepository struct {
	mu    sync.RWMutex
	items map[string]model.Workspace
}

func NewInMemoryWorkspaceRepository() *InMemoryWorkspaceRepository {
	return &InMemoryWorkspaceRepository{items: make(map[string]model.Workspace)}
}

func (r *InMemoryWorkspaceRepository) Create(_ context.Context, w model.Workspace) (model.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.items {
		if existing.DeletedAt == nil && existing.OwnerProfileID == w.OwnerProfileID && existing.Slug == w.Slug {
			return model.Workspace{}, model.NewError(model.ErrorKindSlugAlreadyExists,
				"slug %q đã tồn tại trong workspace của profile %s", w.Slug, w.OwnerProfileID)
		}
	}

	now := time.Now().UTC()
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	w.CreatedAt = now
	w.UpdatedAt = now

	if w.IsDefault {
		r.clearDefaultLocked(w.OwnerProfileID, "", now)
	}
	r.items[w.ID] = w
	return w, nil
}

func (r *InMemoryWorkspaceRepository) Get(_ context.Context, id string) (model.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	w, ok := r.items[id]
	if !ok || w.DeletedAt != nil {
		return model.Workspace{}, fmt.Errorf("%w: workspace %s", model.ErrNotFound, id)
	}
	return w, nil
}

func (r *InMemoryWorkspaceRepository) List(_ context.Context, f model.WorkspaceFilter) ([]model.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Workspace, 0, len(r.items))
	for _, w := range r.items {
		if w.DeletedAt != nil {
			continue
		}
		if f.OwnerProfileID != "" && w.OwnerProfileID != f.OwnerProfileID {
			continue
		}
		if !f.IncludeArchived && w.IsArchived {
			continue
		}
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (r *InMemoryWorkspaceRepository) Update(_ context.Context, id string, upd model.WorkspaceUpdate) (model.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.items[id]
	if !ok || w.DeletedAt != nil {
		return model.Workspace{}, fmt.Errorf("%w: workspace %s", model.ErrNotFound, id)
	}
	now := time.Now().UTC()
	if upd.Name != nil {
		w.Name = *upd.Name
	}
	if upd.Description != nil {
		w.Description = *upd.Description
	}
	if upd.Color != nil {
		w.Color = *upd.Color
	}
	if upd.Icon != nil {
		w.Icon = *upd.Icon
	}
	if upd.Position != nil {
		w.Position = *upd.Position
	}
	if upd.IsArchived != nil {
		w.IsArchived = *upd.IsArchived
	}
	if upd.IsDefault != nil && *upd.IsDefault {
		r.clearDefaultLocked(w.OwnerProfileID, id, now)
		w.IsDefault = true
	} else if upd.IsDefault != nil {
		w.IsDefault = false
	}
	w.UpdatedAt = now
	r.items[id] = w
	return w, nil
}

func (r *InMemoryWorkspaceRepository) clearDefaultLocked(ownerProfileID, exceptID string, now time.Time) {
	for id, existing := range r.items {
		if id == exceptID || existing.DeletedAt != nil {
			continue
		}
		if existing.OwnerProfileID == ownerProfileID && existing.IsDefault && !existing.IsArchived {
			existing.IsDefault = false
			existing.UpdatedAt = now
			r.items[id] = existing
		}
	}
}

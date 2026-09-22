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

type InMemoryTaskRepository struct {
	mu        sync.RWMutex
	items     map[string]model.Task
	logs      []model.StatusLog
	nextLogID int64
}

func NewInMemoryTaskRepository() *InMemoryTaskRepository {
	return &InMemoryTaskRepository{items: make(map[string]model.Task)}
}

func (r *InMemoryTaskRepository) Create(_ context.Context, t model.Task) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t.ParentTaskID != "" {
		if _, ok := r.items[t.ParentTaskID]; !ok {
			return model.Task{}, fmt.Errorf("%w: parent task %s", model.ErrNotFound, t.ParentTaskID)
		}
	}

	now := time.Now().UTC()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Priority == model.TaskPriorityUnspecified {
		t.Priority = model.TaskPriorityMedium
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	r.items[t.ID] = t
	return t, nil
}

func (r *InMemoryTaskRepository) Get(_ context.Context, id string) (model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.items[id]
	if !ok {
		return model.Task{}, fmt.Errorf("%w: task %s", model.ErrNotFound, id)
	}
	return t, nil
}

func (r *InMemoryTaskRepository) List(_ context.Context, f model.TaskFilter) ([]model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Task, 0, len(r.items))
	for _, t := range r.items {
		if f.ProfileID != "" && t.ProfileID != f.ProfileID {
			continue
		}
		if f.WorkspaceID != "" && t.WorkspaceID != f.WorkspaceID {
			continue
		}
		if f.RootOnly {
			if t.ParentTaskID != "" {
				continue
			}
		} else if f.ParentTaskID != "" && t.ParentTaskID != f.ParentTaskID {
			continue
		}
		if f.StatusID != "" && t.StatusID != f.StatusID {
			continue
		}
		if !f.IncludeArchived && t.IsArchived {
			continue
		}
		out = append(out, t)
	}
	sortTasks(out)
	return out, nil
}

func (r *InMemoryTaskRepository) ListChildren(_ context.Context, parentTaskID string) ([]model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Task, 0)
	for _, t := range r.items {
		if t.ParentTaskID == parentTaskID {
			out = append(out, t)
		}
	}
	sortTasks(out)
	return out, nil
}

func (r *InMemoryTaskRepository) Update(_ context.Context, id string, upd model.TaskUpdate) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.items[id]
	if !ok {
		return model.Task{}, fmt.Errorf("%w: task %s", model.ErrNotFound, id)
	}

	if upd.ParentTaskID != nil && *upd.ParentTaskID != t.ParentTaskID {
		newParent := *upd.ParentTaskID
		if newParent == id {
			return model.Task{}, fmt.Errorf("%w: task không thể là cha của chính nó", model.ErrCycle)
		}
		if newParent != "" {
			if _, ok := r.items[newParent]; !ok {
				return model.Task{}, fmt.Errorf("%w: parent task %s", model.ErrNotFound, newParent)
			}
			if r.wouldCycle(id, newParent) {
				return model.Task{}, fmt.Errorf("%w: %s là con/cháu của %s", model.ErrCycle, newParent, id)
			}
		}
		t.ParentTaskID = newParent
	}
	if upd.Title != nil {
		t.Title = *upd.Title
	}
	if upd.Description != nil {
		t.Description = *upd.Description
	}
	if upd.Priority != nil {
		t.Priority = *upd.Priority
	}
	if upd.Position != nil {
		t.Position = *upd.Position
	}
	if upd.StartAt != nil {
		t.StartAt = upd.StartAt
	}
	if upd.DueAt != nil {
		t.DueAt = upd.DueAt
	}
	if upd.IsArchived != nil {
		t.IsArchived = *upd.IsArchived
	}
	t.UpdatedAt = time.Now().UTC()
	r.items[id] = t
	return t, nil
}

func (r *InMemoryTaskRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("%w: task %s", model.ErrNotFound, id)
	}

	delete(r.items, id)
	return nil
}

func (r *InMemoryTaskRepository) ChangeStatus(_ context.Context, id, statusID string, completedAt *time.Time) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.items[id]
	if !ok {
		return model.Task{}, fmt.Errorf("%w: task %s", model.ErrNotFound, id)
	}
	t.StatusID = statusID
	t.CompletedAt = completedAt
	t.UpdatedAt = time.Now().UTC()
	r.items[id] = t
	return t, nil
}

func (r *InMemoryTaskRepository) CountByStatus(_ context.Context, statusID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, t := range r.items {
		if t.StatusID == statusID {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryTaskRepository) AppendStatusLog(_ context.Context, log model.StatusLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextLogID++
	log.ID = r.nextLogID
	if log.ChangedAt.IsZero() {
		log.ChangedAt = time.Now().UTC()
	}
	r.logs = append(r.logs, log)
	return nil
}

func (r *InMemoryTaskRepository) ListStatusLogs(_ context.Context, taskID string) ([]model.StatusLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.StatusLog, 0)
	for _, l := range r.logs {
		if l.TaskID == taskID {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChangedAt.Before(out[j].ChangedAt) })
	return out, nil
}

func (r *InMemoryTaskRepository) wouldCycle(taskID, newParentID string) bool {
	current := newParentID
	for current != "" {
		if current == taskID {
			return true
		}
		parent, ok := r.items[current]
		if !ok {
			return false
		}
		current = parent.ParentTaskID
	}
	return false
}

func sortTasks(items []model.Task) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Position != items[j].Position {
			return items[i].Position < items[j].Position
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}

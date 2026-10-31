package repository

import (
	"cmp"
	"context"
	"fmt"
	"slices"
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
		if f.ProjectID != "" && t.ProjectID != f.ProjectID {
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
		if !matchesListExtensions(t, f) {
			continue
		}
		out = append(out, t)
	}
	sortTasksBy(out, f.SortField, f.SortDesc)
	return paginate(out, f.Limit, f.Offset), nil
}

func matchesListExtensions(t model.Task, f model.TaskFilter) bool {
	if (f.DueFrom != nil || f.DueTo != nil) && t.DueAt == nil {
		return false
	}
	if f.DueFrom != nil && t.DueAt.Before(*f.DueFrom) {
		return false
	}
	if f.DueTo != nil && !t.DueAt.Before(*f.DueTo) {
		return false
	}
	if f.HasDueAt != nil && *f.HasDueAt != (t.DueAt != nil) {
		return false
	}
	if len(f.Priorities) > 0 && !slices.Contains(f.Priorities, t.Priority) {
		return false
	}
	if slices.Contains(f.ExcludeStatusIDs, t.StatusID) {
		return false
	}
	return true
}

func paginate(items []model.Task, limit, offset int) []model.Task {
	if offset >= len(items) {
		if offset > 0 {
			return []model.Task{}
		}
		return items
	}
	items = items[offset:]
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
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

func (r *InMemoryTaskRepository) SetProjectForSubtree(_ context.Context, rootID, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	root, ok := r.items[rootID]
	if !ok {
		return fmt.Errorf("%w: task %s", model.ErrNotFound, rootID)
	}

	now := time.Now().UTC()
	queue := []string{rootID}
	visited := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true

		t := r.items[id]
		t.ProjectID = projectID
		t.UpdatedAt = now
		r.items[id] = t

		for _, c := range r.items {
			if c.ParentTaskID == id && c.ProfileID == root.ProfileID {
				queue = append(queue, c.ID)
			}
		}
	}
	return nil
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

func sortTasksBy(items []model.Task, field model.TaskSortField, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if field == model.TaskSortDueAt && (a.DueAt == nil) != (b.DueAt == nil) {
			return a.DueAt != nil
		}
		if c := compareTaskField(a, b, field); c != 0 {
			if desc {
				return c > 0
			}
			return c < 0
		}
		return a.ID < b.ID
	})
}

func compareTaskField(a, b model.Task, field model.TaskSortField) int {
	switch field {
	case model.TaskSortDueAt:
		if a.DueAt == nil || b.DueAt == nil {
			return 0
		}
		return a.DueAt.Compare(*b.DueAt)
	case model.TaskSortPriority:
		return cmp.Compare(a.Priority, b.Priority)
	case model.TaskSortCreatedAt:
		return a.CreatedAt.Compare(b.CreatedAt)
	case model.TaskSortUpdatedAt:
		return a.UpdatedAt.Compare(b.UpdatedAt)
	default:
		if c := cmp.Compare(a.Position, b.Position); c != 0 {
			return c
		}
		return a.CreatedAt.Compare(b.CreatedAt)
	}
}

func sortTasks(items []model.Task) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Position != items[j].Position {
			return items[i].Position < items[j].Position
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}

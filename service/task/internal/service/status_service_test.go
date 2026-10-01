package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"taskmanager/service/task/internal/model"
)

type stubStatusRepo struct {
	items map[string]model.Status
	seq   int
}

func newStubStatusRepo() *stubStatusRepo {
	return &stubStatusRepo{items: map[string]model.Status{}}
}

func (r *stubStatusRepo) Create(_ context.Context, s model.Status) (model.Status, error) {
	for _, existing := range r.items {
		if existing.ProfileID == s.ProfileID && existing.Slug == s.Slug {
			return model.Status{}, model.ErrAlreadyExists
		}
	}
	r.seq++
	s.ID = "status-" + strconv.Itoa(r.seq)
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = s.CreatedAt
	r.items[s.ID] = s
	return s, nil
}

func (r *stubStatusRepo) Get(_ context.Context, id string) (model.Status, error) {
	s, ok := r.items[id]
	if !ok {
		return model.Status{}, model.ErrNotFound
	}
	return s, nil
}

func (r *stubStatusRepo) List(_ context.Context, profileID string, includeArchived bool) ([]model.Status, error) {
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
	return out, nil
}

func (r *stubStatusRepo) Update(_ context.Context, id string, upd model.StatusUpdate) (model.Status, error) {
	s, ok := r.items[id]
	if !ok {
		return model.Status{}, model.ErrNotFound
	}
	if upd.IsArchived != nil {
		s.IsArchived = *upd.IsArchived
	}
	r.items[id] = s
	return s, nil
}

func (r *stubStatusRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

type stubTransitionRepo struct {
	items map[string]model.Transition
	seq   int
}

func newStubTransitionRepo() *stubTransitionRepo {
	return &stubTransitionRepo{items: map[string]model.Transition{}}
}

func (r *stubTransitionRepo) Create(_ context.Context, t model.Transition) (model.Transition, error) {
	for _, existing := range r.items {
		if existing.ProfileID == t.ProfileID &&
			existing.FromStatusID == t.FromStatusID &&
			existing.ToStatusID == t.ToStatusID {
			return model.Transition{}, model.ErrAlreadyExists
		}
	}
	r.seq++
	t.ID = "transition-" + strconv.Itoa(r.seq)
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	r.items[t.ID] = t
	return t, nil
}

func (r *stubTransitionRepo) Get(_ context.Context, id string) (model.Transition, error) {
	t, ok := r.items[id]
	if !ok {
		return model.Transition{}, model.ErrNotFound
	}
	return t, nil
}

func (r *stubTransitionRepo) List(_ context.Context, profileID, fromStatusID string) ([]model.Transition, error) {
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
	return out, nil
}

func (r *stubTransitionRepo) Update(_ context.Context, _ string, _ model.TransitionUpdate) (model.Transition, error) {
	return model.Transition{}, nil
}

func (r *stubTransitionRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (r *stubTransitionRepo) IsAllowed(_ context.Context, profileID, fromStatusID, toStatusID string) (bool, error) {
	for _, t := range r.items {
		if t.ProfileID == profileID && t.FromStatusID == fromStatusID && t.ToStatusID == toStatusID {
			return t.IsActive, nil
		}
	}
	return false, nil
}

type stubTaskRepo struct{}

func (stubTaskRepo) Create(_ context.Context, t model.Task) (model.Task, error) { return t, nil }
func (stubTaskRepo) Get(_ context.Context, _ string) (model.Task, error) {
	return model.Task{}, model.ErrNotFound
}
func (stubTaskRepo) List(_ context.Context, _ model.TaskFilter) ([]model.Task, error) {
	return nil, nil
}
func (stubTaskRepo) ListChildren(_ context.Context, _ string) ([]model.Task, error) {
	return nil, nil
}
func (stubTaskRepo) Update(_ context.Context, _ string, _ model.TaskUpdate) (model.Task, error) {
	return model.Task{}, nil
}
func (stubTaskRepo) Delete(_ context.Context, _ string) error { return nil }
func (stubTaskRepo) ChangeStatus(_ context.Context, _, _ string, _ *time.Time) (model.Task, error) {
	return model.Task{}, nil
}
func (stubTaskRepo) CountByStatus(_ context.Context, _ string) (int, error) { return 0, nil }
func (stubTaskRepo) AppendStatusLog(_ context.Context, _ model.StatusLog) error {
	return nil
}
func (stubTaskRepo) ListStatusLogs(_ context.Context, _ string) ([]model.StatusLog, error) {
	return nil, nil
}

func newTestStatusService() (TaskStatusService, *stubStatusRepo, *stubTransitionRepo) {
	statuses := newStubStatusRepo()
	transitions := newStubTransitionRepo()
	return NewTaskStatusService(statuses, stubTaskRepo{}, transitions), statuses, transitions
}

func TestSeedDefaultStatusesCreatesFullSet(t *testing.T) {
	svc, _, _ := newTestStatusService()

	statuses, transitions, err := svc.SeedDefaultStatuses(context.Background(), "profile-1")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(statuses) != len(seedDefaultStatuses) {
		t.Fatalf("phải tạo %d status, nhận %d", len(seedDefaultStatuses), len(statuses))
	}
	if len(transitions) != len(seedDefaultTransitionPaths) {
		t.Fatalf("phải tạo %d transition, nhận %d", len(seedDefaultTransitionPaths), len(transitions))
	}
	for _, s := range statuses {
		if s.ProfileID != "profile-1" {
			t.Fatalf("status phải thuộc profile-1, nhận %q", s.ProfileID)
		}
	}
	var defaults int
	for _, s := range statuses {
		if s.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatalf("phải có đúng 1 status default, nhận %d", defaults)
	}
}

func TestSeedDefaultStatusesIsIdempotent(t *testing.T) {
	svc, statusRepo, transitionRepo := newTestStatusService()
	ctx := context.Background()

	if _, _, err := svc.SeedDefaultStatuses(ctx, "profile-1"); err != nil {
		t.Fatalf("seed lần đầu: %v", err)
	}
	if _, _, err := svc.SeedDefaultStatuses(ctx, "profile-1"); err != nil {
		t.Fatalf("seed lần hai: %v", err)
	}

	if len(statusRepo.items) != len(seedDefaultStatuses) {
		t.Fatalf("không được nhân đôi status, nhận %d", len(statusRepo.items))
	}
	if len(transitionRepo.items) != len(seedDefaultTransitionPaths) {
		t.Fatalf("không được nhân đôi transition, nhận %d", len(transitionRepo.items))
	}
}

func TestSeedDefaultStatusesFillsMissingPieces(t *testing.T) {
	svc, statusRepo, transitionRepo := newTestStatusService()
	ctx := context.Background()

	if _, err := statusRepo.Create(ctx, model.Status{
		ProfileID: "profile-1",
		Name:      "To Do",
		Slug:      "todo",
		Category:  model.TaskStatusCategoryTodo,
	}); err != nil {
		t.Fatalf("tạo status trước: %v", err)
	}

	if _, _, err := svc.SeedDefaultStatuses(ctx, "profile-1"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if len(statusRepo.items) != len(seedDefaultStatuses) {
		t.Fatalf("phải bổ sung đủ %d status, nhận %d", len(seedDefaultStatuses), len(statusRepo.items))
	}
	if len(transitionRepo.items) != len(seedDefaultTransitionPaths) {
		t.Fatalf("phải tạo %d transition, nhận %d", len(seedDefaultTransitionPaths), len(transitionRepo.items))
	}
}

func TestSeedDefaultStatusesScopedPerProfile(t *testing.T) {
	svc, statusRepo, _ := newTestStatusService()
	ctx := context.Background()

	if _, _, err := svc.SeedDefaultStatuses(ctx, "profile-1"); err != nil {
		t.Fatalf("seed profile-1: %v", err)
	}
	if _, _, err := svc.SeedDefaultStatuses(ctx, "profile-2"); err != nil {
		t.Fatalf("seed profile-2: %v", err)
	}

	if len(statusRepo.items) != 2*len(seedDefaultStatuses) {
		t.Fatalf("mỗi profile phải có bộ riêng, nhận %d", len(statusRepo.items))
	}
}

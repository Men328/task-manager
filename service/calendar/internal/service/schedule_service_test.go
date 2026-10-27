package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/calendar/internal/model"
)

type fakeScheduleRepository struct {
	items map[string]model.Schedule
}

func newFakeScheduleRepository(items ...model.Schedule) *fakeScheduleRepository {
	repo := &fakeScheduleRepository{items: make(map[string]model.Schedule)}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (r *fakeScheduleRepository) Create(_ context.Context, s model.Schedule) (model.Schedule, error) {
	if s.ID == "" {
		s.ID = "generated"
	}
	r.items[s.ID] = s
	return s, nil
}

func (r *fakeScheduleRepository) Get(_ context.Context, id string) (model.Schedule, error) {
	s, ok := r.items[id]
	if !ok {
		return model.Schedule{}, model.ErrNotFound
	}
	return s, nil
}

func (r *fakeScheduleRepository) List(_ context.Context, f model.ScheduleFilter) ([]model.Schedule, error) {
	out := make([]model.Schedule, 0, len(r.items))
	for _, item := range r.items {
		if f.ProjectID != "" && item.ProjectID != f.ProjectID {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeScheduleRepository) Update(_ context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error) {
	s, ok := r.items[id]
	if !ok {
		return model.Schedule{}, model.ErrNotFound
	}
	if upd.ProjectID != nil {
		s.ProjectID = *upd.ProjectID
	}
	if upd.StartAt != nil {
		s.StartAt = *upd.StartAt
	}
	if upd.EndAt != nil {
		s.EndAt = upd.EndAt
	}
	r.items[id] = s
	return s, nil
}

func (r *fakeScheduleRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

type fakeProjectChecker struct {
	assignable map[string]error
}

func newFakeProjectChecker() *fakeProjectChecker {
	return &fakeProjectChecker{assignable: make(map[string]error)}
}

func (c *fakeProjectChecker) CheckAssignable(_ context.Context, _, projectID string) error {
	return c.assignable[projectID]
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func timePtr(t time.Time) *time.Time { return &t }

func TestCreateRejectsEndBeforeStart(t *testing.T) {
	repo := newFakeScheduleRepository()
	svc := NewScheduleService(repo, newFakeProjectChecker())
	start := mustTime(t, "2026-10-05T02:00:00Z")

	_, err := svc.Create(context.Background(), model.Schedule{
		ProfileID: "p1",
		ProjectID: "proj1",
		Title:     "Sai giờ",
		StartAt:   start,
		EndAt:     timePtr(start.Add(-time.Hour)),
	})
	if err == nil {
		t.Fatal("expected error for end before start")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindTimeRangeInvalid {
		t.Fatalf("expected time_range_invalid, got %v", err)
	}
}

func TestCreateAcceptsValidRange(t *testing.T) {
	repo := newFakeScheduleRepository()
	svc := NewScheduleService(repo, newFakeProjectChecker())
	start := mustTime(t, "2026-10-05T01:00:00Z")

	created, err := svc.Create(context.Background(), model.Schedule{
		ProfileID: "p1",
		ProjectID: "proj1",
		Title:     "Họp nhóm",
		StartAt:   start,
		EndAt:     timePtr(start.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected an id")
	}
}

func TestUpdateValidatesMergedRange(t *testing.T) {
	start := mustTime(t, "2026-10-05T01:00:00Z")
	end := start.Add(time.Hour)
	repo := newFakeScheduleRepository(model.Schedule{
		ID:        "s1",
		ProfileID: "p1",
		Title:     "Cũ",
		StartAt:   start,
		EndAt:     timePtr(end),
	})
	svc := NewScheduleService(repo, newFakeProjectChecker())
	newStart := start.Add(2 * time.Hour)

	_, err := svc.Update(context.Background(), "s1", model.ScheduleUpdate{StartAt: timePtr(newStart)})
	if err == nil {
		t.Fatal("expected error when new start moves past existing end")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindTimeRangeInvalid {
		t.Fatalf("expected time_range_invalid, got %v", err)
	}
}

func TestUpdateMovesBothBounds(t *testing.T) {
	start := mustTime(t, "2026-10-05T01:00:00Z")
	repo := newFakeScheduleRepository(model.Schedule{
		ID:        "s1",
		ProfileID: "p1",
		Title:     "Cũ",
		StartAt:   start,
		EndAt:     timePtr(start.Add(time.Hour)),
	})
	svc := NewScheduleService(repo, newFakeProjectChecker())
	newStart := start.Add(24 * time.Hour)
	newEnd := newStart.Add(2 * time.Hour)

	updated, err := svc.Update(context.Background(), "s1", model.ScheduleUpdate{
		StartAt: timePtr(newStart),
		EndAt:   timePtr(newEnd),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.EndAt == nil || !updated.EndAt.Equal(newEnd) {
		t.Fatalf("expected end moved to %v, got %v", newEnd, updated.EndAt)
	}
}

func TestGetPropagatesNotFound(t *testing.T) {
	svc := NewScheduleService(newFakeScheduleRepository(), newFakeProjectChecker())

	_, err := svc.Get(context.Background(), "missing")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func kindOf(t *testing.T, err error) model.ErrorKind {
	t.Helper()
	var domainErr *model.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected domain error, got %v", err)
	}
	return domainErr.Kind
}

func TestCreateRequiresProject(t *testing.T) {
	repo := newFakeScheduleRepository()
	svc := NewScheduleService(repo, newFakeProjectChecker())

	_, err := svc.Create(context.Background(), model.Schedule{
		ProfileID: "p1", Title: "x", StartAt: mustTime(t, "2026-10-05T01:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
		t.Fatalf("expected project_required, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

func TestCreateRejectsArchivedProject(t *testing.T) {
	repo := newFakeScheduleRepository()
	checker := newFakeProjectChecker()
	checker.assignable["proj1"] = model.NewError(model.ErrorKindProjectArchived, "archived")
	svc := NewScheduleService(repo, checker)

	_, err := svc.Create(context.Background(), model.Schedule{
		ProfileID: "p1", ProjectID: "proj1", Title: "x", StartAt: mustTime(t, "2026-10-05T01:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectArchived {
		t.Fatalf("expected project_archived, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

func TestUpdateMovesScheduleToProject(t *testing.T) {
	repo := newFakeScheduleRepository(model.Schedule{ID: "s1", ProfileID: "p1", ProjectID: "p1proj", Title: "x"})
	svc := NewScheduleService(repo, newFakeProjectChecker())
	target := "p2"

	if _, err := svc.Update(context.Background(), "s1", model.ScheduleUpdate{ProjectID: &target}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.items["s1"].ProjectID != "p2" {
		t.Fatalf("expected project p2, got %s", repo.items["s1"].ProjectID)
	}
}

func TestUpdateRejectsUnassignableProject(t *testing.T) {
	repo := newFakeScheduleRepository(model.Schedule{ID: "s1", ProfileID: "p1", ProjectID: "p1proj", Title: "x"})
	checker := newFakeProjectChecker()
	checker.assignable["p2"] = model.NewError(model.ErrorKindProjectNotFound, "not found")
	svc := NewScheduleService(repo, checker)
	target := "p2"

	_, err := svc.Update(context.Background(), "s1", model.ScheduleUpdate{ProjectID: &target})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectNotFound {
		t.Fatalf("expected project_not_found, got %v", err)
	}
	if repo.items["s1"].ProjectID != "p1proj" {
		t.Fatalf("project must be unchanged, got %s", repo.items["s1"].ProjectID)
	}
}

func TestListFiltersByProject(t *testing.T) {
	repo := newFakeScheduleRepository(
		model.Schedule{ID: "a", ProfileID: "p1", ProjectID: "p1"},
		model.Schedule{ID: "b", ProfileID: "p1", ProjectID: "p2"},
	)
	svc := NewScheduleService(repo, newFakeProjectChecker())

	got, err := svc.List(context.Background(), model.ScheduleFilter{ProfileID: "p1", ProjectID: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("expected only schedule a, got %+v", got)
	}
}

func TestCreateRejectsBlankProject(t *testing.T) {
	repo := newFakeScheduleRepository()
	svc := NewScheduleService(repo, newFakeProjectChecker())

	_, err := svc.Create(context.Background(), model.Schedule{
		ProfileID: "p1", ProjectID: "   ", Title: "x", StartAt: mustTime(t, "2026-10-05T01:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
		t.Fatalf("expected project_required, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

func TestUpdateRejectsEmptyProject(t *testing.T) {
	start := mustTime(t, "2026-10-05T01:00:00Z")
	repo := newFakeScheduleRepository(model.Schedule{ID: "s1", ProfileID: "p1", ProjectID: "p1proj", Title: "x", StartAt: start})
	svc := NewScheduleService(repo, newFakeProjectChecker())

	for _, value := range []string{"", "  "} {
		empty := value
		_, err := svc.Update(context.Background(), "s1", model.ScheduleUpdate{ProjectID: &empty})
		if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
			t.Fatalf("project_id %q: expected project_required, got %v", value, err)
		}
	}
	if repo.items["s1"].ProjectID != "p1proj" {
		t.Fatalf("project must be unchanged, got %s", repo.items["s1"].ProjectID)
	}
}

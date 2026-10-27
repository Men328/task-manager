package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/event/internal/model"
)

type fakeEventRepository struct {
	items map[string]model.Event
}

func newFakeEventRepository(items ...model.Event) *fakeEventRepository {
	repo := &fakeEventRepository{items: make(map[string]model.Event)}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (r *fakeEventRepository) Create(_ context.Context, e model.Event) (model.Event, error) {
	if e.ID == "" {
		e.ID = "generated"
	}
	r.items[e.ID] = e
	return e, nil
}

func (r *fakeEventRepository) Get(_ context.Context, id string) (model.Event, error) {
	e, ok := r.items[id]
	if !ok {
		return model.Event{}, model.ErrNotFound
	}
	return e, nil
}

func (r *fakeEventRepository) List(_ context.Context, f model.Filter) ([]model.Event, error) {
	out := make([]model.Event, 0, len(r.items))
	for _, item := range r.items {
		if f.ProjectID != "" && item.ProjectID != f.ProjectID {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeEventRepository) Update(_ context.Context, id string, upd model.Update) (model.Event, error) {
	e, ok := r.items[id]
	if !ok {
		return model.Event{}, model.ErrNotFound
	}
	if upd.ProjectID != nil {
		e.ProjectID = *upd.ProjectID
	}
	if upd.StartAt != nil {
		e.StartAt = *upd.StartAt
	}
	if upd.EndAt != nil {
		e.EndAt = upd.EndAt
	}
	if upd.Status != nil {
		e.Status = *upd.Status
	}
	r.items[id] = e
	return e, nil
}

func (r *fakeEventRepository) Delete(_ context.Context, id string) error {
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

func mustEventTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func eventTimePtr(t time.Time) *time.Time { return &t }

func TestEventCreateDefaultsStatusToPlanned(t *testing.T) {
	svc := NewEventService(newFakeEventRepository(), newFakeProjectChecker())

	created, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1",
		ProjectID: "proj1",
		Title:     "Hội thảo",
		StartAt:   mustEventTime(t, "2026-11-12T07:00:00Z"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != model.StatusPlanned {
		t.Fatalf("expected planned, got %q", created.Status)
	}
}

func TestEventCreateRejectsEndBeforeStart(t *testing.T) {
	svc := NewEventService(newFakeEventRepository(), newFakeProjectChecker())
	start := mustEventTime(t, "2026-11-12T07:00:00Z")

	_, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1",
		ProjectID: "proj1",
		Title:     "Sai giờ",
		StartAt:   start,
		EndAt:     eventTimePtr(start.Add(-time.Hour)),
	})
	if err == nil {
		t.Fatal("expected error for end before start")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindTimeRangeInvalid {
		t.Fatalf("expected time_range_invalid, got %v", err)
	}
}

func TestEventCreateRejectsUnknownStatus(t *testing.T) {
	svc := NewEventService(newFakeEventRepository(), newFakeProjectChecker())

	_, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1",
		ProjectID: "proj1",
		Title:     "Lạ",
		StartAt:   mustEventTime(t, "2026-11-12T07:00:00Z"),
		Status:    model.Status("BOGUS"),
	})
	if err == nil {
		t.Fatal("expected error for unknown status")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindStatusInvalid {
		t.Fatalf("expected status_invalid, got %v", err)
	}
}

func TestEventUpdateRejectsUnknownStatus(t *testing.T) {
	start := mustEventTime(t, "2026-11-12T07:00:00Z")
	repo := newFakeEventRepository(model.Event{
		ID:        "e1",
		ProfileID: "p1",
		Title:     "Cũ",
		StartAt:   start,
		Status:    model.StatusConfirmed,
	})
	svc := NewEventService(repo, newFakeProjectChecker())
	bogus := model.Status("BOGUS")

	_, err := svc.Update(context.Background(), "e1", model.Update{Status: &bogus})
	if err == nil {
		t.Fatal("expected error for unknown status on update")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindStatusInvalid {
		t.Fatalf("expected status_invalid, got %v", err)
	}
}

func TestEventGetPropagatesNotFound(t *testing.T) {
	svc := NewEventService(newFakeEventRepository(), newFakeProjectChecker())

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
	repo := newFakeEventRepository()
	svc := NewEventService(repo, newFakeProjectChecker())

	_, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1", Title: "x", StartAt: mustEventTime(t, "2026-11-12T07:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
		t.Fatalf("expected project_required, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

func TestCreateRejectsArchivedProject(t *testing.T) {
	repo := newFakeEventRepository()
	checker := newFakeProjectChecker()
	checker.assignable["proj1"] = model.NewError(model.ErrorKindProjectArchived, "archived")
	svc := NewEventService(repo, checker)

	_, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1", ProjectID: "proj1", Title: "x", StartAt: mustEventTime(t, "2026-11-12T07:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectArchived {
		t.Fatalf("expected project_archived, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

func TestUpdateMovesEventToProject(t *testing.T) {
	repo := newFakeEventRepository(model.Event{ID: "e1", ProfileID: "p1", ProjectID: "p1proj", Title: "x"})
	svc := NewEventService(repo, newFakeProjectChecker())
	target := "p2"

	if _, err := svc.Update(context.Background(), "e1", model.Update{ProjectID: &target}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.items["e1"].ProjectID != "p2" {
		t.Fatalf("expected project p2, got %s", repo.items["e1"].ProjectID)
	}
}

func TestUpdateRejectsUnassignableProject(t *testing.T) {
	repo := newFakeEventRepository(model.Event{ID: "e1", ProfileID: "p1", ProjectID: "p1proj", Title: "x"})
	checker := newFakeProjectChecker()
	checker.assignable["p2"] = model.NewError(model.ErrorKindProjectNotFound, "not found")
	svc := NewEventService(repo, checker)
	target := "p2"

	_, err := svc.Update(context.Background(), "e1", model.Update{ProjectID: &target})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectNotFound {
		t.Fatalf("expected project_not_found, got %v", err)
	}
	if repo.items["e1"].ProjectID != "p1proj" {
		t.Fatalf("project must be unchanged, got %s", repo.items["e1"].ProjectID)
	}
}

func TestUpdateRejectsEmptyProject(t *testing.T) {
	repo := newFakeEventRepository(model.Event{ID: "e1", ProfileID: "p1", ProjectID: "p1proj", Title: "x"})
	svc := NewEventService(repo, newFakeProjectChecker())
	empty := "  "

	_, err := svc.Update(context.Background(), "e1", model.Update{ProjectID: &empty})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
		t.Fatalf("expected project_required, got %v", err)
	}
	if repo.items["e1"].ProjectID != "p1proj" {
		t.Fatalf("project must be unchanged, got %s", repo.items["e1"].ProjectID)
	}
}

func TestListFiltersByProject(t *testing.T) {
	repo := newFakeEventRepository(
		model.Event{ID: "a", ProfileID: "p1", ProjectID: "p1"},
		model.Event{ID: "b", ProfileID: "p1", ProjectID: "p2"},
	)
	svc := NewEventService(repo, newFakeProjectChecker())

	got, err := svc.List(context.Background(), model.Filter{ProfileID: "p1", ProjectID: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("expected only event a, got %+v", got)
	}
}

func TestCreateRejectsBlankProject(t *testing.T) {
	repo := newFakeEventRepository()
	svc := NewEventService(repo, newFakeProjectChecker())

	_, err := svc.Create(context.Background(), model.Event{
		ProfileID: "p1", ProjectID: "  ", Title: "x", StartAt: mustEventTime(t, "2026-11-12T07:00:00Z"),
	})
	if err == nil || kindOf(t, err) != model.ErrorKindProjectRequired {
		t.Fatalf("expected project_required, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatal("repo must stay empty")
	}
}

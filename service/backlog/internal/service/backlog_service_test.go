package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/service/backlog/internal/model"
)

type fakeBacklogRepository struct {
	items map[string]model.Item
}

func newFakeBacklogRepository(items ...model.Item) *fakeBacklogRepository {
	repo := &fakeBacklogRepository{items: make(map[string]model.Item)}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (r *fakeBacklogRepository) Create(_ context.Context, item model.Item) (model.Item, error) {
	if item.ID == "" {
		item.ID = "generated"
	}
	r.items[item.ID] = item
	return item, nil
}

func (r *fakeBacklogRepository) Get(_ context.Context, id string) (model.Item, error) {
	item, ok := r.items[id]
	if !ok {
		return model.Item{}, model.ErrNotFound
	}
	return item, nil
}

func (r *fakeBacklogRepository) List(_ context.Context, _ model.Filter) ([]model.Item, error) {
	out := make([]model.Item, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeBacklogRepository) Update(_ context.Context, id string, upd model.Update) (model.Item, error) {
	item, ok := r.items[id]
	if !ok {
		return model.Item{}, model.ErrNotFound
	}
	if upd.Status != nil {
		item.Status = *upd.Status
	}
	if upd.Reason != nil {
		item.Reason = *upd.Reason
	}
	r.items[id] = item
	return item, nil
}

func (r *fakeBacklogRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func TestBacklogCreateDefaultsStatusAndCategory(t *testing.T) {
	svc := NewBacklogService(newFakeBacklogRepository())

	created, err := svc.Create(context.Background(), model.Item{
		ProfileID: "p1",
		Title:     "Newsletter",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != model.StatusNew {
		t.Fatalf("expected NEW, got %q", created.Status)
	}
	if created.Category != model.CategoryOther {
		t.Fatalf("expected other, got %q", created.Category)
	}
}

func TestBacklogCreateKeepsExplicitStatus(t *testing.T) {
	svc := NewBacklogService(newFakeBacklogRepository())

	created, err := svc.Create(context.Background(), model.Item{
		ProfileID: "p1",
		Title:     "Đã xử lý",
		Status:    model.StatusTriaged,
		Category:  "task",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != model.StatusTriaged || created.Category != "task" {
		t.Fatalf("explicit values must be kept, got %+v", created)
	}
}

func TestBacklogCreateRejectsUnknownStatus(t *testing.T) {
	svc := NewBacklogService(newFakeBacklogRepository())

	_, err := svc.Create(context.Background(), model.Item{
		ProfileID: "p1",
		Title:     "Lạ",
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

func TestBacklogUpdateRejectsUnknownStatus(t *testing.T) {
	repo := newFakeBacklogRepository(model.Item{
		ID:        "b1",
		ProfileID: "p1",
		Title:     "Cũ",
		Status:    model.StatusNew,
		Category:  model.CategoryOther,
	})
	svc := NewBacklogService(repo)
	bogus := model.Status("BOGUS")

	_, err := svc.Update(context.Background(), "b1", model.Update{Status: &bogus})
	if err == nil {
		t.Fatal("expected error for unknown status on update")
	}

	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != model.ErrorKindStatusInvalid {
		t.Fatalf("expected status_invalid, got %v", err)
	}
}

func TestBacklogGetPropagatesNotFound(t *testing.T) {
	svc := NewBacklogService(newFakeBacklogRepository())

	_, err := svc.Get(context.Background(), "missing")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

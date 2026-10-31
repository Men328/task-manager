package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"taskmanager/service/project/internal/model"
)

type fakeProjectRepository struct {
	items map[string]model.Project
	seq   int
}

func newFakeProjectRepository() *fakeProjectRepository {
	return &fakeProjectRepository{items: make(map[string]model.Project)}
}

func (r *fakeProjectRepository) Create(_ context.Context, p model.Project) (model.Project, error) {
	for _, it := range r.items {
		if it.ProfileID == p.ProfileID && strings.EqualFold(it.Name, p.Name) {
			return model.Project{}, model.NewError(model.ErrorKindNameDuplicate, "duplicate")
		}
	}
	r.seq++
	p.ID = "p" + strconv.Itoa(r.seq)
	r.items[p.ID] = p
	return p, nil
}

func (r *fakeProjectRepository) Get(_ context.Context, id string) (model.Project, error) {
	p, ok := r.items[id]
	if !ok {
		return model.Project{}, model.ErrNotFound
	}
	return p, nil
}

func (r *fakeProjectRepository) List(_ context.Context, f model.ProjectFilter) ([]model.Project, error) {
	out := []model.Project{}
	for _, it := range r.items {
		if it.ProfileID == f.ProfileID && (f.IncludeArchived || !it.IsArchived) {
			out = append(out, it)
		}
	}
	return out, nil
}

func (r *fakeProjectRepository) GetDefault(_ context.Context, profileID string) (model.Project, error) {
	for _, it := range r.items {
		if it.ProfileID == profileID && it.IsDefault {
			return it, nil
		}
	}
	return model.Project{}, model.ErrNotFound
}

func (r *fakeProjectRepository) Update(_ context.Context, id string, upd model.ProjectUpdate) (model.Project, error) {
	p, ok := r.items[id]
	if !ok {
		return model.Project{}, model.ErrNotFound
	}
	if upd.Name != nil {
		p.Name = *upd.Name
	}
	if upd.Description != nil {
		p.Description = *upd.Description
	}
	if upd.Color != nil {
		p.Color = *upd.Color
	}
	r.items[id] = p
	return p, nil
}

func (r *fakeProjectRepository) SetArchived(_ context.Context, id string, archived bool) (model.Project, error) {
	p, ok := r.items[id]
	if !ok {
		return model.Project{}, model.ErrNotFound
	}
	p.IsArchived = archived
	r.items[id] = p
	return p, nil
}

func requireKind(t *testing.T, err error, kind model.ErrorKind) {
	t.Helper()
	var de *model.Error
	if !errors.As(err, &de) || de.Kind != kind {
		t.Fatalf("expected error kind %q, got %v", kind, err)
	}
}

func TestSeedDefaultIsIdempotent(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepository())
	a, err := svc.SeedDefault(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.SeedDefault(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID || a.Name != "Chung" || !a.IsDefault {
		t.Fatalf("unexpected: %+v %+v", a, b)
	}
}

func TestListCreatesDefaultWhenMissing(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepository())
	items, err := svc.List(context.Background(), model.ProjectFilter{ProfileID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].IsDefault {
		t.Fatalf("unexpected: %+v", items)
	}
}

func TestArchiveDefaultRejected(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepository())
	d, _ := svc.SeedDefault(context.Background(), "p1")
	_, err := svc.Archive(context.Background(), d.ID)
	requireKind(t, err, model.ErrorKindDefaultNotArchivable)
}

func TestArchiveAndUnarchive(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	p, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "Web"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Archive(ctx, p.ID)
	if err != nil || !got.IsArchived {
		t.Fatalf("archive: %+v %v", got, err)
	}
	items, _ := svc.List(ctx, model.ProjectFilter{ProfileID: "p1"})
	for _, it := range items {
		if it.ID == p.ID {
			t.Fatal("archived project should be hidden")
		}
	}
	got, err = svc.Unarchive(ctx, p.ID)
	if err != nil || got.IsArchived {
		t.Fatalf("unarchive: %+v %v", got, err)
	}
}

func TestCreateTrimsAndRequiresName(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	if _, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "  "}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	p, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "  Web  "})
	if err != nil || p.Name != "Web" {
		t.Fatalf("unexpected: %+v %v", p, err)
	}
}

func TestCreateNeverMakesDefault(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepository())
	p, err := svc.Create(context.Background(), model.Project{ProfileID: "p1", Name: "X", IsDefault: true})
	if err != nil || p.IsDefault {
		t.Fatalf("unexpected: %+v %v", p, err)
	}
}

func TestUpdateArchivedProjectRejected(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	p, _ := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "Web"})
	_, _ = svc.Archive(ctx, p.ID)
	n := "New"
	_, err := svc.Update(ctx, p.ID, model.ProjectUpdate{Name: &n})
	requireKind(t, err, model.ErrorKindArchived)
}

func TestCreateValidatesNameLength(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	// 100 ký tự có dấu (rune) vẫn hợp lệ dù vượt 100 byte.
	ok := strings.Repeat("ữ", 100)
	if _, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: ok}); err != nil {
		t.Fatalf("100 rune phải hợp lệ, nhận %v", err)
	}
	_, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: strings.Repeat("a", 101)})
	requireKind(t, err, model.ErrorKindNameTooLong)
}

func TestCreateValidatesColor(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	for i, color := range []string{"", "#aabbcc", "#AABBCC80"} {
		if _, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "Ok" + strconv.Itoa(i), Color: color}); err != nil {
			t.Fatalf("màu %q phải hợp lệ, nhận %v", color, err)
		}
	}
	for i, color := range []string{"red", "#abc", "#aabbccd", "aabbcc", "#gggggg"} {
		_, err := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "Bad" + strconv.Itoa(i), Color: color})
		requireKind(t, err, model.ErrorKindColorInvalid)
	}
}

func TestUpdateValidatesNameAndColor(t *testing.T) {
	ctx := context.Background()
	svc := NewProjectService(newFakeProjectRepository())
	p, _ := svc.Create(ctx, model.Project{ProfileID: "p1", Name: "Web"})

	long := strings.Repeat("b", 101)
	_, err := svc.Update(ctx, p.ID, model.ProjectUpdate{Name: &long})
	requireKind(t, err, model.ErrorKindNameTooLong)

	bad := "#12345"
	_, err = svc.Update(ctx, p.ID, model.ProjectUpdate{Color: &bad})
	requireKind(t, err, model.ErrorKindColorInvalid)

	empty := ""
	if _, err := svc.Update(ctx, p.ID, model.ProjectUpdate{Color: &empty}); err != nil {
		t.Fatalf("màu rỗng phải hợp lệ, nhận %v", err)
	}
}

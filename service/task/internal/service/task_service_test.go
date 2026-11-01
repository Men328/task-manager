package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"taskmanager/service/task/internal/model"
)

// ------------------------------------------------------------------ fakes

type fakeTaskRepo struct {
	items      map[string]model.Task
	nextID     int
	listResult []model.Task
	lastFilter model.TaskFilter
}

func newFakeTaskRepo() *fakeTaskRepo { return &fakeTaskRepo{items: map[string]model.Task{}} }

func (r *fakeTaskRepo) put(t model.Task) { r.items[t.ID] = t }

func (r *fakeTaskRepo) Create(_ context.Context, t model.Task) (model.Task, error) {
	if t.ID == "" {
		r.nextID++
		t.ID = fmt.Sprintf("t%d", r.nextID)
	}
	r.items[t.ID] = t
	return t, nil
}

func (r *fakeTaskRepo) Get(_ context.Context, id string) (model.Task, error) {
	t, ok := r.items[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	return t, nil
}

func (r *fakeTaskRepo) List(_ context.Context, f model.TaskFilter) ([]model.Task, error) {
	r.lastFilter = f
	out := r.listResult
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (r *fakeTaskRepo) ListChildren(_ context.Context, parentTaskID string) ([]model.Task, error) {
	out := []model.Task{}
	for _, t := range r.items {
		if t.ParentTaskID == parentTaskID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *fakeTaskRepo) Update(_ context.Context, id string, upd model.TaskUpdate) (model.Task, error) {
	t, ok := r.items[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	if upd.ProjectID != nil {
		return model.Task{}, errors.New("project changes must go through SetProjectForSubtree")
	}
	if upd.ParentTaskID != nil {
		t.ParentTaskID = *upd.ParentTaskID
	}
	if upd.Title != nil {
		t.Title = *upd.Title
	}
	if upd.ChangesReminder() {
		t.RemindAt = upd.RemindAt
		t.RemindedAt = upd.RemindedAt
	}
	r.items[id] = t
	return t, nil
}

func (r *fakeTaskRepo) SetProjectForSubtree(_ context.Context, rootID, projectID string) error {
	t, ok := r.items[rootID]
	if !ok {
		return model.ErrNotFound
	}
	t.ProjectID = projectID
	r.items[rootID] = t
	for _, c := range r.items {
		if c.ParentTaskID == rootID {
			if err := r.SetProjectForSubtree(context.Background(), c.ID, projectID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *fakeTaskRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}

func (r *fakeTaskRepo) ChangeStatus(context.Context, string, string, *time.Time) (model.Task, error) {
	return model.Task{}, nil
}

func (r *fakeTaskRepo) CountByStatus(context.Context, string) (int, error) { return 0, nil }

func (r *fakeTaskRepo) AppendStatusLog(context.Context, model.StatusLog) error { return nil }

func (r *fakeTaskRepo) ListStatusLogs(context.Context, string) ([]model.StatusLog, error) {
	return nil, nil
}

type fakeTaskStatusRepo struct{ StatusRepository }

func (fakeTaskStatusRepo) List(_ context.Context, profileID string, _ bool) ([]model.Status, error) {
	return []model.Status{
		{ID: "st-default", ProfileID: profileID, IsDefault: true, Category: model.TaskStatusCategoryTodo},
		{ID: "st-done", ProfileID: profileID, Category: model.TaskStatusCategoryDone},
		{ID: "st-cancelled", ProfileID: profileID, Category: model.TaskStatusCategoryCancelled, IsArchived: true},
	}, nil
}

type fakeTaskTransitionRepo struct{ TransitionRepository }

type fakeProjectChecker struct {
	err   error
	calls []string
}

func (f *fakeProjectChecker) CheckAssignable(_ context.Context, _, projectID string) error {
	f.calls = append(f.calls, projectID)
	return f.err
}

func newTestTaskService(repo *fakeTaskRepo, checker *fakeProjectChecker) TaskService {
	return NewTaskService(repo, fakeTaskStatusRepo{}, fakeTaskTransitionRepo{}, checker)
}

func strPtr(s string) *string { return &s }

func wantKind(t *testing.T, err error, kind model.ErrorKind) {
	t.Helper()
	var de *model.Error
	if !errors.As(err, &de) || de.Kind != kind {
		t.Fatalf("want error kind %s, got %v", kind, err)
	}
}

// seedTree tạo cha(p1) -> con(p1) -> cháu(p1).
func seedTree(repo *fakeTaskRepo) {
	repo.put(model.Task{ID: "parent", ProfileID: "u1", ProjectID: "p1"})
	repo.put(model.Task{ID: "child", ProfileID: "u1", ProjectID: "p1", ParentTaskID: "parent"})
	repo.put(model.Task{ID: "grandchild", ProfileID: "u1", ProjectID: "p1", ParentTaskID: "child"})
}

// ------------------------------------------------------------------ tests

func TestCreateTaskRequiresProject(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	_, err := svc.Create(context.Background(), model.Task{ProfileID: "u1", Title: "x"})
	wantKind(t, err, model.ErrorKindProjectRequired)
	if len(repo.items) != 0 {
		t.Fatalf("task must not be created")
	}
}

func TestCreateTaskChecksProject(t *testing.T) {
	repo := newFakeTaskRepo()
	checker := &fakeProjectChecker{err: model.NewError(model.ErrorKindProjectArchived, "archived")}
	svc := newTestTaskService(repo, checker)

	_, err := svc.Create(context.Background(), model.Task{ProfileID: "u1", ProjectID: "p1", Title: "x"})
	wantKind(t, err, model.ErrorKindProjectArchived)
	if len(repo.items) != 0 {
		t.Fatalf("task must not be created")
	}

	checker.err = nil
	created, err := svc.Create(context.Background(), model.Task{ProfileID: "u1", ProjectID: "p1", Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ProjectID != "p1" {
		t.Fatalf("want project p1, got %q", created.ProjectID)
	}
}

func TestCreateSubtaskInheritsParentProject(t *testing.T) {
	for _, requested := range []string{"p2", ""} {
		t.Run("requested="+requested, func(t *testing.T) {
			repo := newFakeTaskRepo()
			repo.put(model.Task{ID: "parent", ProfileID: "u1", ProjectID: "p1"})
			svc := newTestTaskService(repo, &fakeProjectChecker{})

			child, err := svc.Create(context.Background(), model.Task{
				ProfileID: "u1", ParentTaskID: "parent", ProjectID: requested, Title: "c",
			})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if child.ProjectID != "p1" {
				t.Fatalf("want inherited project p1, got %q", child.ProjectID)
			}
		})
	}
}

func TestUpdateParentProjectMovesSubtree(t *testing.T) {
	repo := newFakeTaskRepo()
	seedTree(repo)
	checker := &fakeProjectChecker{}
	svc := newTestTaskService(repo, checker)

	updated, err := svc.Update(context.Background(), "parent", model.TaskUpdate{ProjectID: strPtr("p2")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ProjectID != "p2" {
		t.Fatalf("returned task: want p2, got %q", updated.ProjectID)
	}
	for _, id := range []string{"parent", "child", "grandchild"} {
		if got := repo.items[id].ProjectID; got != "p2" {
			t.Fatalf("%s: want p2, got %q", id, got)
		}
	}
	if len(checker.calls) != 1 || checker.calls[0] != "p2" {
		t.Fatalf("want CheckAssignable(p2) once, got %v", checker.calls)
	}
}

func TestUpdateSubtaskProjectRejected(t *testing.T) {
	repo := newFakeTaskRepo()
	seedTree(repo)
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	_, err := svc.Update(context.Background(), "child", model.TaskUpdate{ProjectID: strPtr("p2")})
	wantKind(t, err, model.ErrorKindSubtaskProjectLocked)
	if got := repo.items["child"].ProjectID; got != "p1" {
		t.Fatalf("child project must stay p1, got %q", got)
	}

	// Gửi lại đúng project của cha thì không lỗi.
	if _, err := svc.Update(context.Background(), "child", model.TaskUpdate{ProjectID: strPtr("p1")}); err != nil {
		t.Fatalf("same-as-parent project should be accepted: %v", err)
	}
}

func TestUpdateTaskRejectsEmptyProject(t *testing.T) {
	repo := newFakeTaskRepo()
	seedTree(repo)
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	_, err := svc.Update(context.Background(), "parent", model.TaskUpdate{ProjectID: strPtr("  ")})
	wantKind(t, err, model.ErrorKindProjectRequired)
}

func TestReparentTakesNewParentProject(t *testing.T) {
	repo := newFakeTaskRepo()
	repo.put(model.Task{ID: "x", ProfileID: "u1", ProjectID: "p1"})
	repo.put(model.Task{ID: "x-child", ProfileID: "u1", ProjectID: "p1", ParentTaskID: "x"})
	repo.put(model.Task{ID: "other", ProfileID: "u1", ProjectID: "p2"})
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	updated, err := svc.Update(context.Background(), "x", model.TaskUpdate{
		ParentTaskID: strPtr("other"),
		Title:        strPtr("renamed"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ProjectID != "p2" || updated.ParentTaskID != "other" || updated.Title != "renamed" {
		t.Fatalf("unexpected returned task: %+v", updated)
	}
	for _, id := range []string{"x", "x-child"} {
		if got := repo.items[id].ProjectID; got != "p2" {
			t.Fatalf("%s: want p2, got %q", id, got)
		}
	}
}

func TestUpdateTaskRejectsArchivedProject(t *testing.T) {
	repo := newFakeTaskRepo()
	seedTree(repo)
	checker := &fakeProjectChecker{err: model.NewError(model.ErrorKindProjectArchived, "archived")}
	svc := newTestTaskService(repo, checker)

	_, err := svc.Update(context.Background(), "parent", model.TaskUpdate{
		ProjectID: strPtr("p2"),
		Title:     strPtr("changed"),
	})
	wantKind(t, err, model.ErrorKindProjectArchived)
	for _, id := range []string{"parent", "child", "grandchild"} {
		if got := repo.items[id].ProjectID; got != "p1" {
			t.Fatalf("%s: want p1, got %q", id, got)
		}
	}
	if repo.items["parent"].Title != "" {
		t.Fatalf("title must not change")
	}
}

func TestReparentIntoArchivedProjectRejected(t *testing.T) {
	repo := newFakeTaskRepo()
	repo.put(model.Task{ID: "x", ProfileID: "u1", ProjectID: "p1", Title: "orig"})
	repo.put(model.Task{ID: "x-child", ProfileID: "u1", ProjectID: "p1", ParentTaskID: "x"})
	repo.put(model.Task{ID: "other", ProfileID: "u1", ProjectID: "p2"})
	checker := &fakeProjectChecker{err: model.NewError(model.ErrorKindProjectArchived, "archived")}
	svc := newTestTaskService(repo, checker)

	_, err := svc.Update(context.Background(), "x", model.TaskUpdate{
		ParentTaskID: strPtr("other"),
		Title:        strPtr("renamed"),
	})
	wantKind(t, err, model.ErrorKindProjectArchived)
	if len(checker.calls) != 1 || checker.calls[0] != "p2" {
		t.Fatalf("want CheckAssignable(p2) once, got %v", checker.calls)
	}
	x := repo.items["x"]
	if x.ParentTaskID != "" || x.ProjectID != "p1" || x.Title != "orig" {
		t.Fatalf("x must be unchanged, got %+v", x)
	}
	if got := repo.items["x-child"].ProjectID; got != "p1" {
		t.Fatalf("x-child: want p1, got %q", got)
	}
}

func TestCreateSubtaskUnderArchivedProjectRejected(t *testing.T) {
	repo := newFakeTaskRepo()
	repo.put(model.Task{ID: "parent", ProfileID: "u1", ProjectID: "p1"})
	checker := &fakeProjectChecker{err: model.NewError(model.ErrorKindProjectArchived, "archived")}
	svc := newTestTaskService(repo, checker)

	_, err := svc.Create(context.Background(), model.Task{ProfileID: "u1", ParentTaskID: "parent", Title: "c"})
	wantKind(t, err, model.ErrorKindProjectArchived)
	if len(checker.calls) != 1 || checker.calls[0] != "p1" {
		t.Fatalf("want CheckAssignable(p1) once, got %v", checker.calls)
	}
	if len(repo.items) != 1 {
		t.Fatalf("subtask must not be created, repo has %d items", len(repo.items))
	}
}

func TestListOpenOnlyExcludesDoneAndCancelledStatuses(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	if _, err := svc.List(context.Background(), model.TaskFilter{ProfileID: "u1", OpenOnly: true}); err != nil {
		t.Fatalf("List: %v", err)
	}
	got := repo.lastFilter.ExcludeStatusIDs
	if len(got) != 2 || got[0] != "st-done" || got[1] != "st-cancelled" {
		t.Fatalf("ExcludeStatusIDs = %v, want [st-done st-cancelled]", got)
	}
}

func TestListWithoutOpenOnlyKeepsAllStatuses(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})

	if _, err := svc.List(context.Background(), model.TaskFilter{ProfileID: "u1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(repo.lastFilter.ExcludeStatusIDs) != 0 {
		t.Fatalf("ExcludeStatusIDs = %v, want empty", repo.lastFilter.ExcludeStatusIDs)
	}
}

func TestListPagination(t *testing.T) {
	repo := newFakeTaskRepo()
	for i := range 5 {
		repo.listResult = append(repo.listResult, model.Task{ID: fmt.Sprintf("t%d", i)})
	}
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	ctx := context.Background()

	page, err := svc.List(ctx, model.TaskFilter{ProfileID: "u1", Limit: 3, Offset: 6})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.lastFilter.Limit != 4 {
		t.Fatalf("repo Limit = %d, want page size + 1 = 4", repo.lastFilter.Limit)
	}
	if len(page.Tasks) != 3 || page.NextOffset != 9 {
		t.Fatalf("got %d tasks, NextOffset %d; want 3 tasks, NextOffset 9", len(page.Tasks), page.NextOffset)
	}

	repo.listResult = repo.listResult[:2]
	page, _ = svc.List(ctx, model.TaskFilter{ProfileID: "u1", Limit: 3})
	if len(page.Tasks) != 2 || page.NextOffset != 0 {
		t.Fatalf("last page: got %d tasks, NextOffset %d; want 2, 0", len(page.Tasks), page.NextOffset)
	}

	page, _ = svc.List(ctx, model.TaskFilter{ProfileID: "u1"})
	if repo.lastFilter.Limit != 0 || page.NextOffset != 0 {
		t.Fatalf("no paging: repo Limit %d, NextOffset %d; want 0, 0", repo.lastFilter.Limit, page.NextOffset)
	}
}

func TestCreateTaskReminderState(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	created, err := svc.Create(ctx, model.Task{ProfileID: "u1", ProjectID: "p1", Title: "a", RemindAt: &future})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.RemindedAt != nil {
		t.Fatalf("future reminder must stay pending, got reminded_at %v", created.RemindedAt)
	}

	created, _ = svc.Create(ctx, model.Task{ProfileID: "u1", ProjectID: "p1", Title: "b", RemindAt: &past})
	if created.RemindedAt == nil {
		t.Fatal("past reminder must be stored as already reminded so it never fires")
	}
}

func TestUpdateTaskReminderResetsAndClears(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	ctx := context.Background()
	sent := time.Now().Add(-time.Minute)
	old := time.Now().Add(-2 * time.Minute)
	repo.put(model.Task{ID: "t1", ProfileID: "u1", ProjectID: "p1", RemindAt: &old, RemindedAt: &sent})

	future := time.Now().Add(time.Hour)
	updated, err := svc.Update(ctx, "t1", model.TaskUpdate{RemindAt: &future})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.RemindAt == nil || !updated.RemindAt.Equal(future) || updated.RemindedAt != nil {
		t.Fatalf("new reminder time must re-arm the reminder, got %+v", updated)
	}

	updated, _ = svc.Update(ctx, "t1", model.TaskUpdate{ClearRemindAt: true})
	if updated.RemindAt != nil || updated.RemindedAt != nil {
		t.Fatalf("clear must remove the reminder, got %+v", updated)
	}

	title := "renamed"
	repo.put(model.Task{ID: "t2", ProfileID: "u1", ProjectID: "p1", RemindAt: &old, RemindedAt: &sent})
	updated, _ = svc.Update(ctx, "t2", model.TaskUpdate{Title: &title})
	if updated.RemindedAt == nil {
		t.Fatal("editing other fields must not re-arm a sent reminder")
	}
}

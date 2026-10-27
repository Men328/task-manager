package repository

import (
	"context"
	"errors"
	"testing"

	"taskmanager/service/task/internal/model"
)

func TestInMemorySetProjectForSubtree(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryTaskRepository()
	mk := func(id, parent, project string) {
		if _, err := repo.Create(ctx, model.Task{ID: id, ProfileID: "u1", ParentTaskID: parent, ProjectID: project}); err != nil {
			t.Fatalf("Create %s: %v", id, err)
		}
	}
	mk("root", "", "p1")
	mk("child", "root", "p1")
	mk("grandchild", "child", "p1")
	mk("other", "", "p1")

	if err := repo.SetProjectForSubtree(ctx, "child", "p2"); err != nil {
		t.Fatalf("SetProjectForSubtree: %v", err)
	}
	want := map[string]string{"root": "p1", "child": "p2", "grandchild": "p2", "other": "p1"}
	for id, project := range want {
		got, _ := repo.Get(ctx, id)
		if got.ProjectID != project {
			t.Fatalf("%s: want %s, got %q", id, project, got.ProjectID)
		}
	}

	list, _ := repo.List(ctx, model.TaskFilter{ProfileID: "u1", ProjectID: "p2"})
	if len(list) != 2 {
		t.Fatalf("filter by project p2: want 2 tasks, got %d", len(list))
	}

	if err := repo.SetProjectForSubtree(ctx, "missing", "p2"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("missing root: want ErrNotFound, got %v", err)
	}
}

package repository

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"taskmanager/service/task/internal/model"
)

func seedListTasks(t *testing.T) *InMemoryTaskRepository {
	t.Helper()
	ctx := context.Background()
	repo := NewInMemoryTaskRepository()
	day := func(d int) *time.Time {
		v := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC)
		return &v
	}
	tasks := []model.Task{
		{ID: "a", DueAt: day(1), Priority: model.TaskPriorityLow, StatusID: "open", Position: 3},
		{ID: "b", DueAt: day(2), Priority: model.TaskPriorityUrgent, StatusID: "open", Position: 1},
		{ID: "c", DueAt: day(3), Priority: model.TaskPriorityHigh, StatusID: "done", Position: 2},
		{ID: "d", Priority: model.TaskPriorityHigh, StatusID: "open", Position: 0},
		{ID: "e", DueAt: day(5), Priority: model.TaskPriorityMedium, StatusID: "open", Position: 4},
	}
	for _, task := range tasks {
		task.ProfileID = "u1"
		task.ProjectID = "p1"
		if _, err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Create %s: %v", task.ID, err)
		}
	}
	return repo
}

func taskIDs(items []model.Task) []string {
	out := make([]string, 0, len(items))
	for _, t := range items {
		out = append(out, t.ID)
	}
	return out
}

func TestInMemoryListFilters(t *testing.T) {
	ctx := context.Background()
	repo := seedListTasks(t)
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	yes, no := true, false

	cases := []struct {
		name   string
		filter model.TaskFilter
		want   []string
	}{
		{"due range is half-open", model.TaskFilter{DueFrom: &from, DueTo: &to}, []string{"b", "c"}},
		{"due_from only", model.TaskFilter{DueFrom: &from}, []string{"b", "c", "e"}},
		{"has due", model.TaskFilter{HasDueAt: &yes}, []string{"b", "c", "a", "e"}},
		{"no due", model.TaskFilter{HasDueAt: &no}, []string{"d"}},
		{"priorities", model.TaskFilter{Priorities: []model.TaskPriority{model.TaskPriorityHigh, model.TaskPriorityUrgent}}, []string{"d", "b", "c"}},
		{"exclude statuses", model.TaskFilter{ExcludeStatusIDs: []string{"done"}}, []string{"d", "b", "a", "e"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.filter.ProfileID = "u1"
			got, err := repo.List(ctx, tc.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if ids := taskIDs(got); !slices.Equal(ids, tc.want) {
				t.Fatalf("got %v, want %v", ids, tc.want)
			}
		})
	}
}

func TestInMemoryListSort(t *testing.T) {
	ctx := context.Background()
	repo := seedListTasks(t)

	cases := []struct {
		name  string
		field model.TaskSortField
		desc  bool
		want  []string
	}{
		{"default position", model.TaskSortUnspecified, false, []string{"d", "b", "c", "a", "e"}},
		{"due asc, no due last", model.TaskSortDueAt, false, []string{"a", "b", "c", "e", "d"}},
		{"due desc, no due still last", model.TaskSortDueAt, true, []string{"e", "c", "b", "a", "d"}},
		{"priority desc, id tiebreak", model.TaskSortPriority, true, []string{"b", "c", "d", "e", "a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := repo.List(ctx, model.TaskFilter{ProfileID: "u1", SortField: tc.field, SortDesc: tc.desc})
			if ids := taskIDs(got); !slices.Equal(ids, tc.want) {
				t.Fatalf("got %v, want %v", ids, tc.want)
			}
		})
	}
}

func TestInMemoryListPaginationCoversAllWithoutDuplicates(t *testing.T) {
	ctx := context.Background()
	repo := seedListTasks(t)

	var seen []string
	for offset := 0; ; offset += 2 {
		got, _ := repo.List(ctx, model.TaskFilter{ProfileID: "u1", Limit: 2, Offset: offset})
		if len(got) == 0 {
			break
		}
		seen = append(seen, taskIDs(got)...)
	}
	if want := []string{"d", "b", "c", "a", "e"}; !slices.Equal(seen, want) {
		t.Fatalf("pages = %v, want %v", seen, want)
	}
}

func TestBuildTaskListQueryLegacyFilter(t *testing.T) {
	query, args := buildTaskListQuery(model.TaskFilter{ProfileID: "u1", RootOnly: true})

	for _, part := range []string{"profile_id = $1::uuid", "parent_task_id IS NULL", "NOT is_archived", "ORDER BY tasks.position ASC, tasks.created_at ASC, tasks.id"} {
		if !strings.Contains(query, part) {
			t.Fatalf("query missing %q:\n%s", part, query)
		}
	}
	if strings.Contains(query, "LIMIT") {
		t.Fatalf("legacy filter must not paginate:\n%s", query)
	}
	if len(args) != 1 {
		t.Fatalf("args = %v, want only profile id", args)
	}
}

func TestTaskOrderByUsesTableColumns(t *testing.T) {
	order := taskOrderBy(model.TaskSortPriority, true)
	if order != "tasks.priority DESC, tasks.id" {
		t.Fatalf("order = %q: bare priority would resolve to the priority::text output column and sort alphabetically", order)
	}
}

func TestBuildTaskListQueryExtensions(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	query, args := buildTaskListQuery(model.TaskFilter{
		ProfileID:        "u1",
		DueFrom:          &from,
		Priorities:       []model.TaskPriority{model.TaskPriorityUrgent},
		ExcludeStatusIDs: []string{"s1"},
		SortField:        model.TaskSortDueAt,
		SortDesc:         true,
		Limit:            11,
		Offset:           20,
	})

	for _, part := range []string{
		"due_at >= $2",
		"priority = ANY($3::task.task_priority[])",
		"NOT (status_id = ANY($4::uuid[]))",
		"ORDER BY tasks.due_at DESC NULLS LAST, tasks.id",
		"LIMIT $5 OFFSET $6",
	} {
		if !strings.Contains(query, part) {
			t.Fatalf("query missing %q:\n%s", part, query)
		}
	}
	if len(args) != 6 || args[4] != 11 || args[5] != 20 {
		t.Fatalf("args = %v", args)
	}
	if prios, ok := args[2].([]string); !ok || !slices.Equal(prios, []string{"urgent"}) {
		t.Fatalf("priority arg = %#v, want [urgent]", args[2])
	}
}

func TestInMemoryListSearchIgnoresCaseAndAccents(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryTaskRepository()
	for _, task := range []model.Task{
		{ID: "a", Title: "Nộp Báo cáo tuần", Description: "gửi thầy"},
		{ID: "b", Title: "Họp nhóm", Description: "chuẩn bị Đồ án"},
		{ID: "c", Title: "Mua sữa"},
	} {
		task.ProfileID = "u1"
		if _, err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	cases := map[string][]string{
		"bao cao": {"a"},
		"BÁO CÁO": {"a"},
		"do an":   {"b"},
		"thay":    {"a"},
		"xyz":     {},
	}
	for query, want := range cases {
		got, err := repo.List(ctx, model.TaskFilter{ProfileID: "u1", Query: query})
		if err != nil {
			t.Fatalf("List(%q): %v", query, err)
		}
		if ids := taskIDs(got); !slices.Equal(ids, want) {
			t.Fatalf("query %q: got %v, want %v", query, ids, want)
		}
	}
}

func TestBuildTaskListQuerySearch(t *testing.T) {
	query, args := buildTaskListQuery(model.TaskFilter{ProfileID: "u1", Query: "50%_off\\x"})

	for _, part := range []string{
		taskSearchExpr + ` LIKE '%' || task.f_search_text($2) || '%' ESCAPE '\'`,
		"ORDER BY similarity(" + taskSearchExpr + ", task.f_search_text($2)) DESC, tasks.updated_at DESC, tasks.id",
	} {
		if !strings.Contains(query, part) {
			t.Fatalf("query missing %q:\n%s", part, query)
		}
	}
	if args[1] != `50\%\_off\\x` {
		t.Fatalf("query arg = %q, want LIKE wildcards escaped", args[1])
	}

	query, _ = buildTaskListQuery(model.TaskFilter{ProfileID: "u1", Query: "abc", SortField: model.TaskSortDueAt})
	if !strings.Contains(query, "ORDER BY tasks.due_at ASC NULLS LAST") {
		t.Fatalf("explicit sort must win over similarity:\n%s", query)
	}
}

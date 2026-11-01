package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/task/internal/model"
)

var vn = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		panic(err)
	}
	return loc
}()

func at(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, vn)
}

func TestNormalizeRecurrence(t *testing.T) {
	due := at(2026, 11, 6, 0, 0)

	rule, tz, err := normalizeRecurrence(" freq=weekly;byday=fr ", "", &due)
	if err != nil || rule != "FREQ=WEEKLY;BYDAY=FR" || tz != "Asia/Ho_Chi_Minh" {
		t.Fatalf("got %q %q %v", rule, tz, err)
	}
	if rule, _, err := normalizeRecurrence("", "", nil); err != nil || rule != "" {
		t.Fatalf("empty rule must mean no recurrence, got %q %v", rule, err)
	}

	for name, tc := range map[string]struct {
		rule string
		tz   string
		due  *time.Time
	}{
		"no due date":        {"FREQ=DAILY", "", nil},
		"unsupported freq":   {"FREQ=HOURLY", "", &due},
		"too large gap":      {"FREQ=DAILY;INTERVAL=400", "", &due},
		"byhour not allowed": {"FREQ=DAILY;BYHOUR=9", "", &due},
		"garbage":            {"every day", "", &due},
		"bad timezone":       {"FREQ=DAILY", "Mars/Base", &due},
	} {
		if _, _, err := normalizeRecurrence(tc.rule, tc.tz, tc.due); !isKind(err, model.ErrorKindRecurrenceInvalid) {
			t.Fatalf("%s: want recurrence_invalid, got %v", name, err)
		}
	}
}

func isKind(err error, kind model.ErrorKind) bool {
	var domainErr *model.Error
	return errors.As(err, &domainErr) && domainErr.Kind == kind
}

func TestNextOccurrence(t *testing.T) {
	cases := []struct {
		name      string
		rule      string
		due       time.Time
		completed time.Time
		want      time.Time
		wantRule  string
		ends      bool
	}{
		{"daily done on time", "FREQ=DAILY", at(2026, 11, 2, 0, 0), at(2026, 11, 2, 10, 0), at(2026, 11, 3, 0, 0), "FREQ=DAILY", false},
		{"daily missed 10 days skips to tomorrow", "FREQ=DAILY", at(2026, 10, 23, 0, 0), at(2026, 11, 2, 10, 0), at(2026, 11, 3, 0, 0), "FREQ=DAILY", false},
		{"keeps the time of day", "FREQ=DAILY", at(2026, 11, 2, 9, 0), at(2026, 11, 2, 8, 0), at(2026, 11, 3, 9, 0), "FREQ=DAILY", false},
		{"weekly friday done early", "FREQ=WEEKLY;BYDAY=FR", at(2026, 11, 6, 0, 0), at(2026, 11, 4, 9, 0), at(2026, 11, 13, 0, 0), "FREQ=WEEKLY;BYDAY=FR", false},
		{"weekdays from friday jumps to monday", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", at(2026, 11, 6, 0, 0), at(2026, 11, 6, 9, 0), at(2026, 11, 9, 0, 0), "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", false},
		{"every 2 days", "FREQ=DAILY;INTERVAL=2", at(2026, 11, 2, 0, 0), at(2026, 11, 2, 9, 0), at(2026, 11, 4, 0, 0), "FREQ=DAILY;INTERVAL=2", false},
		{"monthly 31st skips 30-day months", "FREQ=MONTHLY;BYMONTHDAY=31", at(2026, 10, 31, 0, 0), at(2026, 10, 31, 9, 0), at(2026, 12, 31, 0, 0), "FREQ=MONTHLY;BYMONTHDAY=31", false},
		{"count decrements", "FREQ=DAILY;COUNT=3", at(2026, 11, 2, 0, 0), at(2026, 11, 2, 9, 0), at(2026, 11, 3, 0, 0), "FREQ=DAILY;COUNT=2", false},
		{"count 1 ends the series", "FREQ=DAILY;COUNT=1", at(2026, 11, 2, 0, 0), at(2026, 11, 2, 9, 0), time.Time{}, "", true},
		{"until ends the series", "FREQ=WEEKLY;UNTIL=20261110T000000Z", at(2026, 11, 6, 0, 0), at(2026, 11, 6, 9, 0), time.Time{}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			occ, ok, err := nextOccurrence(tc.rule, "Asia/Ho_Chi_Minh", tc.due.UTC(), tc.completed.UTC())
			if err != nil {
				t.Fatalf("nextOccurrence: %v", err)
			}
			if tc.ends {
				if ok {
					t.Fatalf("series must end, got %v", occ.due.In(vn))
				}
				return
			}
			if !ok || !occ.due.Equal(tc.want) || occ.rule != tc.wantRule {
				t.Fatalf("got %v %q (ok=%v), want %v %q", occ.due.In(vn), occ.rule, ok, tc.want, tc.wantRule)
			}
		})
	}
}

func recurringTask(repo *fakeTaskRepo, due time.Time, remind *time.Time) model.Task {
	task := model.Task{
		ID: "r1", ProfileID: "u1", ProjectID: "p1", StatusID: "st-default", Title: "Nộp báo cáo",
		Priority: model.TaskPriorityHigh, DueAt: &due, RemindAt: remind,
		RecurrenceRule: "FREQ=DAILY", RecurrenceTZ: "Asia/Ho_Chi_Minh", RecurrenceSeriesID: "r1",
	}
	repo.put(task)
	return task
}

func TestCompletingRecurringTaskSpawnsNextOnce(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	ctx := context.Background()
	due := time.Now().Add(2 * time.Hour).Truncate(time.Minute)
	remind := due.Add(-24 * time.Hour)
	recurringTask(repo, due, &remind)

	change, err := svc.ChangeStatus(ctx, "r1", "st-done", "")
	if err != nil || change.NextErr != nil || change.Next == nil {
		t.Fatalf("ChangeStatus: %+v, %v", change, err)
	}
	next := *change.Next
	if !next.DueAt.Equal(due.Add(24*time.Hour)) || next.StatusID != "st-default" || next.Title != "Nộp báo cáo" ||
		next.Priority != model.TaskPriorityHigh || next.RecurrenceSeriesID != "r1" || next.RecurrenceRule != "FREQ=DAILY" {
		t.Fatalf("next task = %+v", next)
	}
	if next.RemindAt == nil || !next.RemindAt.Equal(due) {
		t.Fatalf("reminder must keep its offset to the due date, got %v", next.RemindAt)
	}
	if next.RemindedAt != nil {
		t.Fatal("a future reminder on the new occurrence must be pending")
	}

	if _, err := svc.ChangeStatus(ctx, "r1", "st-default", ""); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	again, err := svc.ChangeStatus(ctx, "r1", "st-done", "")
	if err != nil || again.Next != nil || again.NextErr != nil {
		t.Fatalf("completing again must not spawn a second occurrence: %+v %v", again, err)
	}
	if len(repo.items) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(repo.items))
	}
}

func TestSpawnFailureDoesNotBlockCompletion(t *testing.T) {
	repo := newFakeTaskRepo()
	checker := &fakeProjectChecker{}
	svc := newTestTaskService(repo, checker)
	recurringTask(repo, time.Now().Add(time.Hour), nil)
	checker.err = model.NewError(model.ErrorKindProjectArchived, "archived")

	change, err := svc.ChangeStatus(context.Background(), "r1", "st-done", "")
	if err != nil {
		t.Fatalf("completion must succeed even if the next occurrence cannot be created: %v", err)
	}
	if change.Task.StatusID != "st-done" || change.Next != nil || !isKind(change.NextErr, model.ErrorKindProjectArchived) {
		t.Fatalf("change = %+v", change)
	}
}

func TestCancellingRecurringTaskDoesNotSpawn(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	recurringTask(repo, time.Now().Add(time.Hour), nil)

	change, err := svc.ChangeStatus(context.Background(), "r1", "st-cancelled", "")
	if err != nil || change.Next != nil {
		t.Fatalf("cancel must not spawn: %+v %v", change, err)
	}
}

func TestCreateAndUpdateRecurrence(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newTestTaskService(repo, &fakeProjectChecker{})
	ctx := context.Background()
	due := time.Now().Add(48 * time.Hour)

	created, err := svc.Create(ctx, model.Task{ProfileID: "u1", ProjectID: "p1", Title: "a", DueAt: &due, RecurrenceRule: "freq=weekly"})
	if err != nil || created.RecurrenceRule != "FREQ=WEEKLY" || created.RecurrenceSeriesID != created.ID || created.ID == "" {
		t.Fatalf("created = %+v, %v", created, err)
	}

	_, err = svc.Create(ctx, model.Task{ProfileID: "u1", ProjectID: "p1", Title: "b", RecurrenceRule: "FREQ=DAILY"})
	if !isKind(err, model.ErrorKindRecurrenceInvalid) {
		t.Fatalf("recurrence without due date must fail, got %v", err)
	}

	repo.put(model.Task{ID: "plain", ProfileID: "u1", ProjectID: "p1", DueAt: &due})
	rule := "FREQ=MONTHLY"
	updated, err := svc.Update(ctx, "plain", model.TaskUpdate{RecurrenceRule: &rule})
	if err != nil || updated.RecurrenceRule != "FREQ=MONTHLY" || updated.RecurrenceSeriesID != "plain" || updated.RecurrenceTZ == "" {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	empty := ""
	updated, _ = svc.Update(ctx, "plain", model.TaskUpdate{RecurrenceRule: &empty})
	if updated.RecurrenceRule != "" {
		t.Fatalf("empty rule must stop repeating, got %q", updated.RecurrenceRule)
	}
}

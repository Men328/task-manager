package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/report/internal/model"
)

type fakeDataSource struct {
	statuses []model.Status
	tasks    []model.Task
	err      error
}

func (f *fakeDataSource) FetchStatuses(_ context.Context, _ string) ([]model.Status, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.statuses, nil
}

func (f *fakeDataSource) FetchTasks(_ context.Context, _ string, _ bool) ([]model.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tasks, nil
}

type fakeActivitySource struct {
	events    []model.Event
	schedules []model.Schedule
	backlogs  []model.BacklogItem
	err       error
}

func (f *fakeActivitySource) FetchEvents(_ context.Context, _ string) ([]model.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.events, nil
}

func (f *fakeActivitySource) FetchSchedules(_ context.Context, _ string) ([]model.Schedule, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.schedules, nil
}

func (f *fakeActivitySource) FetchBacklogs(_ context.Context, _ string) ([]model.BacklogItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.backlogs, nil
}

func ts(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed.UTC()
}

func tsPtr(value string) *time.Time {
	converted := ts(value)
	return &converted
}

func fixture() (*fakeDataSource, Clock) {
	statuses := []model.Status{
		{ID: "s1", Name: "Todo", Slug: "todo", Color: "#9aa0ae", Category: model.CategoryTodo, Position: 0},
		{ID: "s2", Name: "Doing", Slug: "doing", Color: "#4c6ef5", Category: model.CategoryInProgress, Position: 1},
		{ID: "s3", Name: "Done", Slug: "done", Color: "#37b24d", Category: model.CategoryDone, Position: 2},
		{ID: "s4", Name: "Cancelled", Slug: "cancelled", Color: "#f06595", Category: model.CategoryCancelled, Position: 3},
	}
	tasks := []model.Task{
		{ID: "t1", StatusID: "s1", DueAt: tsPtr("2025-06-01T00:00:00Z"), CreatedAt: ts("2025-06-10T08:00:00Z")},
		{ID: "t2", StatusID: "s2", DueAt: tsPtr("2025-06-18T00:00:00Z"), CreatedAt: ts("2025-06-11T08:00:00Z")},
		{ID: "t3", StatusID: "s3", CompletedAt: tsPtr("2025-06-13T09:00:00Z"), CreatedAt: ts("2025-06-12T08:00:00Z")},
		{ID: "t4", ParentID: "t3", StatusID: "s1", CreatedAt: ts("2025-06-12T10:00:00Z")},
		{ID: "t5", StatusID: "s4", CompletedAt: tsPtr("2025-06-14T09:00:00Z"), CreatedAt: ts("2025-05-01T08:00:00Z")},
	}
	now := ts("2025-06-15T12:00:00Z")
	return &fakeDataSource{statuses: statuses, tasks: tasks}, func() time.Time { return now }
}

func newService(source TaskDataSource, clock Clock) ReportService {
	return NewReportService(source, &fakeActivitySource{}, clock)
}

func TestOverview(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	overview, err := report.Overview(context.Background(), model.Query{ProfileID: "p1"})
	if err != nil {
		t.Fatalf("Overview trả lỗi: %v", err)
	}

	cases := []struct {
		name string
		got  int32
		want int32
	}{
		{"total", overview.TotalTasks, 4},
		{"root", overview.RootTasks, 3},
		{"subtask", overview.SubtaskCount, 1},
		{"todo", overview.TodoTasks, 2},
		{"in_progress", overview.InProgressTasks, 1},
		{"done", overview.DoneTasks, 1},
		{"cancelled", overview.CancelledTasks, 0},
		{"overdue", overview.OverdueTasks, 1},
		{"due_soon", overview.DueSoonTasks, 1},
		{"created_in_range", overview.CreatedInRange, 4},
		{"completed_in_range", overview.CompletedInRange, 2},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %d, muốn %d", tc.name, tc.got, tc.want)
		}
	}

	if overview.CompletionRate != 25 {
		t.Errorf("completion_rate = %v, muốn 25", overview.CompletionRate)
	}
	if overview.OverdueRate != 25 {
		t.Errorf("overdue_rate = %v, muốn 25", overview.OverdueRate)
	}
	if !overview.From.Equal(ts("2025-05-16T12:00:00Z")) {
		t.Errorf("from mặc định = %v, muốn 2025-05-16T12:00:00Z", overview.From)
	}
	if !overview.To.Equal(ts("2025-06-15T12:00:00Z")) {
		t.Errorf("to mặc định = %v, muốn 2025-06-15T12:00:00Z", overview.To)
	}
}

func TestStatusBreakdown(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	breakdown, err := report.StatusBreakdown(context.Background(), model.Query{ProfileID: "p1"})
	if err != nil {
		t.Fatalf("StatusBreakdown trả lỗi: %v", err)
	}
	if breakdown.Total != 4 {
		t.Fatalf("total = %d, muốn 4", breakdown.Total)
	}
	if len(breakdown.Items) != 4 {
		t.Fatalf("số item = %d, muốn 4", len(breakdown.Items))
	}

	wantCount := map[string]int32{"s1": 2, "s2": 1, "s3": 1, "s4": 0}
	wantPercent := map[string]float64{"s1": 50, "s2": 25, "s3": 25, "s4": 0}
	for _, item := range breakdown.Items {
		if item.Count != wantCount[item.StatusID] {
			t.Errorf("%s count = %d, muốn %d", item.StatusID, item.Count, wantCount[item.StatusID])
		}
		if item.Percentage != wantPercent[item.StatusID] {
			t.Errorf("%s percent = %v, muốn %v", item.StatusID, item.Percentage, wantPercent[item.StatusID])
		}
	}
	if breakdown.Items[0].StatusID != "s1" || breakdown.Items[3].StatusID != "s4" {
		t.Errorf("thứ tự item không theo position: %v", breakdown.Items)
	}
}

func TestTimeSeriesDay(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	from := ts("2025-06-10T00:00:00Z")
	to := ts("2025-06-13T23:59:59Z")
	series, err := report.TimeSeries(context.Background(), model.Query{ProfileID: "p1", From: &from, To: &to}, model.IntervalDay)
	if err != nil {
		t.Fatalf("TimeSeries trả lỗi: %v", err)
	}
	if len(series.Points) != 4 {
		t.Fatalf("số bucket = %d, muốn 4", len(series.Points))
	}
	if series.TotalCreated != 4 {
		t.Errorf("total_created = %d, muốn 4", series.TotalCreated)
	}
	if series.TotalCompleted != 1 {
		t.Errorf("total_completed = %d, muốn 1", series.TotalCompleted)
	}
	if series.Points[0].Bucket != "2025-06-10" || series.Points[0].Created != 1 {
		t.Errorf("bucket[0] = %+v, muốn 2025-06-10/1", series.Points[0])
	}
	if series.Points[2].Created != 2 {
		t.Errorf("bucket[2].created = %d, muốn 2", series.Points[2].Created)
	}
	if series.Points[3].Completed != 1 {
		t.Errorf("bucket[3].completed = %d, muốn 1", series.Points[3].Completed)
	}
}

func TestTimeSeriesWeekStartsMonday(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	from := ts("2025-06-01T00:00:00Z")
	to := ts("2025-06-30T23:59:59Z")
	series, err := report.TimeSeries(context.Background(), model.Query{ProfileID: "p1", From: &from, To: &to}, model.IntervalWeek)
	if err != nil {
		t.Fatalf("TimeSeries trả lỗi: %v", err)
	}
	if series.Points[0].Bucket != "2025-05-26" {
		t.Errorf("bucket đầu = %s, muốn 2025-05-26 (thứ Hai)", series.Points[0].Bucket)
	}
	if series.TotalCompleted != 2 {
		t.Errorf("total_completed = %d, muốn 2", series.TotalCompleted)
	}
}

func TestInvalidRange(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	from := ts("2025-06-20T00:00:00Z")
	to := ts("2025-06-10T00:00:00Z")
	_, err := report.Overview(context.Background(), model.Query{ProfileID: "p1", From: &from, To: &to})
	assertKind(t, err, model.ErrorKindRangeInvalid)
}

func TestInvalidInterval(t *testing.T) {
	source, clock := fixture()
	report := newService(source, clock)

	_, err := report.TimeSeries(context.Background(), model.Query{ProfileID: "p1"}, model.Interval("hour"))
	assertKind(t, err, model.ErrorKindIntervalInvalid)
}

func TestSourceFailure(t *testing.T) {
	clock := func() time.Time { return ts("2025-06-15T12:00:00Z") }
	report := newService(&fakeDataSource{err: errors.New("boom")}, clock)

	_, err := report.Overview(context.Background(), model.Query{ProfileID: "p1"})
	assertKind(t, err, model.ErrorKindSourceFailed)
}

func TestActivity(t *testing.T) {
	_, clock := fixture()
	activity := &fakeActivitySource{
		events: []model.Event{
			{ID: "e1", Status: model.EventStatusPlanned, StartAt: ts("2025-06-12T08:00:00Z")},
			{ID: "e2", Status: model.EventStatusConfirmed, StartAt: ts("2025-07-01T08:00:00Z")},
			{ID: "e3", Status: model.EventStatusCancelled, StartAt: ts("2025-06-01T08:00:00Z"), AllDay: true},
		},
		schedules: []model.Schedule{
			{ID: "s1", StartAt: ts("2025-06-15T00:00:00Z"), AllDay: true},
			{ID: "s2", StartAt: ts("2025-06-20T09:00:00Z")},
			{ID: "s3", StartAt: ts("2025-05-01T09:00:00Z")},
		},
		backlogs: []model.BacklogItem{
			{ID: "b1", Status: model.BacklogStatusNew, Category: "other", CreatedAt: ts("2025-06-10T08:00:00Z")},
			{ID: "b2", Status: model.BacklogStatusTriaged, Category: "task", CreatedAt: ts("2025-06-11T08:00:00Z")},
			{ID: "b3", Status: model.BacklogStatusArchived, Category: "other", CreatedAt: ts("2025-05-01T08:00:00Z")},
			{ID: "b4", Status: model.BacklogStatusNew, Category: "", CreatedAt: ts("2025-06-12T08:00:00Z")},
		},
	}
	report := NewReportService(&fakeDataSource{}, activity, clock)

	got, err := report.Activity(context.Background(), model.Query{ProfileID: "p1"})
	if err != nil {
		t.Fatalf("Activity trả lỗi: %v", err)
	}

	eventCases := []struct {
		name string
		got  int32
		want int32
	}{
		{"events.total", got.Events.TotalEvents, 2},
		{"events.planned", got.Events.PlannedEvents, 1},
		{"events.confirmed", got.Events.ConfirmedEvents, 0},
		{"events.cancelled", got.Events.CancelledEvents, 1},
		{"events.upcoming", got.Events.UpcomingEvents, 1},
		{"events.all_day", got.Events.AllDayEvents, 1},
		{"schedules.total", got.Schedules.TotalSchedules, 1},
		{"schedules.all_day", got.Schedules.AllDaySchedules, 1},
		{"schedules.upcoming", got.Schedules.UpcomingSchedules, 1},
		{"schedules.today", got.Schedules.TodaySchedules, 1},
		{"backlogs.total", got.Backlogs.TotalBacklogs, 3},
		{"backlogs.new", got.Backlogs.NewBacklogs, 2},
		{"backlogs.triaged", got.Backlogs.TriagedBacklogs, 1},
		{"backlogs.archived", got.Backlogs.ArchivedBacklogs, 0},
	}
	for _, tc := range eventCases {
		if tc.got != tc.want {
			t.Errorf("%s = %d, muốn %d", tc.name, tc.got, tc.want)
		}
	}

	if len(got.Backlogs.ByCategory) != 2 {
		t.Fatalf("by_category = %+v, muốn 2 mục", got.Backlogs.ByCategory)
	}
	if got.Backlogs.ByCategory[0].Category != "other" || got.Backlogs.ByCategory[0].Count != 2 {
		t.Errorf("by_category[0] = %+v, muốn other/2", got.Backlogs.ByCategory[0])
	}
	if got.Backlogs.ByCategory[1].Category != "task" || got.Backlogs.ByCategory[1].Count != 1 {
		t.Errorf("by_category[1] = %+v, muốn task/1", got.Backlogs.ByCategory[1])
	}
}

func TestActivitySourceFailure(t *testing.T) {
	clock := func() time.Time { return ts("2025-06-15T12:00:00Z") }
	report := NewReportService(&fakeDataSource{}, &fakeActivitySource{err: errors.New("boom")}, clock)

	_, err := report.Activity(context.Background(), model.Query{ProfileID: "p1"})
	assertKind(t, err, model.ErrorKindSourceFailed)
}

func assertKind(t *testing.T, err error, kind model.ErrorKind) {
	t.Helper()
	var domainErr *model.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("lỗi %v không phải *model.Error", err)
	}
	if domainErr.Kind != kind {
		t.Fatalf("kind = %s, muốn %s", domainErr.Kind, kind)
	}
}

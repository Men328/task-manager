package service

import (
	"context"
	"math"
	"sort"
	"time"

	"taskmanager/service/report/internal/model"
)

const (
	defaultRangeDays = 30
	dueSoonDays      = 7
	maxBuckets       = 5000
)

type reportService struct {
	source   TaskDataSource
	activity ActivityDataSource
	clock    Clock
}

func NewReportService(source TaskDataSource, activity ActivityDataSource, clock Clock) ReportService {
	if clock == nil {
		clock = time.Now
	}
	return &reportService{source: source, activity: activity, clock: clock}
}

func (s *reportService) Overview(ctx context.Context, q model.Query) (model.Overview, error) {
	scope, err := s.resolveScope(q)
	if err != nil {
		return model.Overview{}, err
	}
	data, err := s.load(ctx, q)
	if err != nil {
		return model.Overview{}, err
	}

	now := s.clock().UTC()
	dueSoonLimit := now.AddDate(0, 0, dueSoonDays)

	overview := model.Overview{
		ProfileID: q.ProfileID,
		From:      scope.From,
		To:        scope.To,
	}

	for _, task := range data.tasks {
		if !inRange(task.CreatedAt, scope.From, scope.To) {
			continue
		}
		overview.TotalTasks++
		overview.CreatedInRange++
		if task.ParentID == "" {
			overview.RootTasks++
		} else {
			overview.SubtaskCount++
		}

		status, known := data.byID[task.StatusID]
		switch {
		case !known:
			overview.TodoTasks++
		case status.Category == model.CategoryDone:
			overview.DoneTasks++
		case status.Category == model.CategoryInProgress:
			overview.InProgressTasks++
		case status.Category == model.CategoryCancelled:
			overview.CancelledTasks++
		default:
			overview.TodoTasks++
		}

		if task.DueAt == nil {
			continue
		}
		if known && (status.Category == model.CategoryDone || status.Category == model.CategoryCancelled) {
			continue
		}
		due := task.DueAt.UTC()
		if due.Before(now) {
			overview.OverdueTasks++
		} else if due.Before(dueSoonLimit) {
			overview.DueSoonTasks++
		}
	}

	for _, task := range data.tasks {
		if task.CompletedAt != nil && inRange(*task.CompletedAt, scope.From, scope.To) {
			overview.CompletedInRange++
		}
	}

	if overview.TotalTasks > 0 {
		overview.CompletionRate = round2(float64(overview.DoneTasks) / float64(overview.TotalTasks) * 100)
		overview.OverdueRate = round2(float64(overview.OverdueTasks) / float64(overview.TotalTasks) * 100)
	}

	return overview, nil
}

func (s *reportService) StatusBreakdown(ctx context.Context, q model.Query) (model.StatusBreakdown, error) {
	scope, err := s.resolveScope(q)
	if err != nil {
		return model.StatusBreakdown{}, err
	}
	data, err := s.load(ctx, q)
	if err != nil {
		return model.StatusBreakdown{}, err
	}

	counts := make(map[string]int32, len(data.statuses))
	var total int32
	for _, task := range data.tasks {
		if !inRange(task.CreatedAt, scope.From, scope.To) {
			continue
		}
		counts[task.StatusID]++
		total++
	}

	items := make([]model.StatusBreakdownItem, 0, len(data.statuses))
	for _, status := range data.statuses {
		count := counts[status.ID]
		percentage := 0.0
		if total > 0 {
			percentage = round2(float64(count) / float64(total) * 100)
		}
		items = append(items, model.StatusBreakdownItem{
			StatusID:   status.ID,
			StatusName: status.Name,
			StatusSlug: status.Slug,
			Color:      status.Color,
			Category:   string(status.Category),
			Count:      count,
			Percentage: percentage,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := data.byID[items[i].StatusID]
		right := data.byID[items[j].StatusID]
		if left.Position != right.Position {
			return left.Position < right.Position
		}
		return items[i].StatusName < items[j].StatusName
	})

	return model.StatusBreakdown{Items: items, Total: total}, nil
}

func (s *reportService) TimeSeries(ctx context.Context, q model.Query, interval model.Interval) (model.TimeSeries, error) {
	if interval == "" {
		interval = model.IntervalDay
	}
	if !interval.Valid() {
		return model.TimeSeries{}, model.NewError(model.ErrorKindIntervalInvalid, "unsupported interval %q", interval)
	}

	scope, err := s.resolveScope(q)
	if err != nil {
		return model.TimeSeries{}, err
	}
	data, err := s.load(ctx, q)
	if err != nil {
		return model.TimeSeries{}, err
	}

	buckets := make(map[int64]model.TimeSeriesPoint)
	order := make([]int64, 0)
	for cursor := truncate(scope.From, interval); !cursor.After(scope.To); cursor = addInterval(cursor, interval) {
		if len(order) >= maxBuckets {
			return model.TimeSeries{}, model.NewError(model.ErrorKindRangeInvalid, "time range needs too many buckets")
		}
		key := cursor.UnixNano()
		buckets[key] = model.TimeSeriesPoint{BucketStart: cursor, Bucket: cursor.Format("2006-01-02")}
		order = append(order, key)
	}

	var totalCreated, totalCompleted int32
	for _, task := range data.tasks {
		if inRange(task.CreatedAt, scope.From, scope.To) {
			key := truncate(task.CreatedAt, interval).UnixNano()
			if point, ok := buckets[key]; ok {
				point.Created++
				buckets[key] = point
				totalCreated++
			}
		}
		if task.CompletedAt != nil && inRange(*task.CompletedAt, scope.From, scope.To) {
			key := truncate(*task.CompletedAt, interval).UnixNano()
			if point, ok := buckets[key]; ok {
				point.Completed++
				buckets[key] = point
				totalCompleted++
			}
		}
	}

	points := make([]model.TimeSeriesPoint, 0, len(order))
	for _, key := range order {
		points = append(points, buckets[key])
	}

	return model.TimeSeries{
		Interval:       interval,
		Points:         points,
		TotalCreated:   totalCreated,
		TotalCompleted: totalCompleted,
	}, nil
}

func (s *reportService) Activity(ctx context.Context, q model.Query) (model.ActivityReport, error) {
	scope, err := s.resolveScope(q)
	if err != nil {
		return model.ActivityReport{}, err
	}

	events, err := s.activity.FetchEvents(ctx, q.ProfileID)
	if err != nil {
		return model.ActivityReport{}, model.NewError(model.ErrorKindSourceFailed, "load events: %v", err)
	}
	schedules, err := s.activity.FetchSchedules(ctx, q.ProfileID)
	if err != nil {
		return model.ActivityReport{}, model.NewError(model.ErrorKindSourceFailed, "load schedules: %v", err)
	}
	backlogs, err := s.activity.FetchBacklogs(ctx, q.ProfileID)
	if err != nil {
		return model.ActivityReport{}, model.NewError(model.ErrorKindSourceFailed, "load backlogs: %v", err)
	}

	now := s.clock().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.AddDate(0, 0, 1)

	report := model.ActivityReport{ProfileID: q.ProfileID, From: scope.From, To: scope.To}

	for _, event := range events {
		if event.StartAt.IsZero() {
			continue
		}
		if !event.StartAt.Before(now) {
			report.Events.UpcomingEvents++
		}
		if !overlaps(event.StartAt, event.EndAt, scope.From, scope.To) {
			continue
		}
		report.Events.TotalEvents++
		if event.AllDay {
			report.Events.AllDayEvents++
		}
		switch event.Status {
		case model.EventStatusPlanned:
			report.Events.PlannedEvents++
		case model.EventStatusConfirmed:
			report.Events.ConfirmedEvents++
		case model.EventStatusCancelled:
			report.Events.CancelledEvents++
		}
	}

	for _, schedule := range schedules {
		if schedule.StartAt.IsZero() {
			continue
		}
		if !schedule.StartAt.Before(now) {
			report.Schedules.UpcomingSchedules++
		}
		if !schedule.StartAt.Before(todayStart) && schedule.StartAt.Before(todayEnd) {
			report.Schedules.TodaySchedules++
		}
		if !overlaps(schedule.StartAt, schedule.EndAt, scope.From, scope.To) {
			continue
		}
		report.Schedules.TotalSchedules++
		if schedule.AllDay {
			report.Schedules.AllDaySchedules++
		}
	}

	categoryCounts := make(map[string]int32)
	for _, item := range backlogs {
		if !inRange(item.CreatedAt, scope.From, scope.To) {
			continue
		}
		report.Backlogs.TotalBacklogs++
		switch item.Status {
		case model.BacklogStatusNew:
			report.Backlogs.NewBacklogs++
		case model.BacklogStatusTriaged:
			report.Backlogs.TriagedBacklogs++
		case model.BacklogStatusArchived:
			report.Backlogs.ArchivedBacklogs++
		}
		category := item.Category
		if category == "" {
			category = model.BacklogCategoryOther
		}
		categoryCounts[category]++
	}
	report.Backlogs.ByCategory = sortCategories(categoryCounts)

	return report, nil
}

func sortCategories(counts map[string]int32) []model.CategoryCount {
	out := make([]model.CategoryCount, 0, len(counts))
	for category, count := range counts {
		out = append(out, model.CategoryCount{Category: category, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Category < out[j].Category
	})
	return out
}

func overlaps(start time.Time, end *time.Time, from, to time.Time) bool {
	effectiveEnd := start.UTC()
	if end != nil {
		effectiveEnd = end.UTC()
	}
	if effectiveEnd.Before(from) {
		return false
	}
	return !start.UTC().After(to)
}

type dataset struct {
	statuses []model.Status
	byID     map[string]model.Status
	tasks    []model.Task
}

func (s *reportService) load(ctx context.Context, q model.Query) (dataset, error) {
	statuses, err := s.source.FetchStatuses(ctx, q.ProfileID)
	if err != nil {
		return dataset{}, model.NewError(model.ErrorKindSourceFailed, "load statuses: %v", err)
	}
	tasks, err := s.source.FetchTasks(ctx, q.ProfileID, q.IncludeArchived)
	if err != nil {
		return dataset{}, model.NewError(model.ErrorKindSourceFailed, "load tasks: %v", err)
	}

	byID := make(map[string]model.Status, len(statuses))
	for _, status := range statuses {
		byID[status.ID] = status
	}
	return dataset{statuses: statuses, byID: byID, tasks: tasks}, nil
}

func (s *reportService) resolveScope(q model.Query) (model.Scope, error) {
	now := s.clock().UTC()
	to := now
	if q.To != nil {
		to = q.To.UTC()
	}
	from := to.AddDate(0, 0, -defaultRangeDays)
	if q.From != nil {
		from = q.From.UTC()
	}
	if !from.Before(to) {
		return model.Scope{}, model.NewError(model.ErrorKindRangeInvalid, "from must be before to")
	}
	return model.Scope{ProfileID: q.ProfileID, From: from, To: to}, nil
}

func inRange(value, from, to time.Time) bool {
	value = value.UTC()
	return !value.Before(from) && !value.After(to)
}

func truncate(value time.Time, interval model.Interval) time.Time {
	value = value.UTC()
	switch interval {
	case model.IntervalWeek:
		weekday := int(value.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		day := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
		return day.AddDate(0, 0, -(weekday - 1))
	case model.IntervalMonth:
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func addInterval(value time.Time, interval model.Interval) time.Time {
	switch interval {
	case model.IntervalWeek:
		return value.AddDate(0, 0, 7)
	case model.IntervalMonth:
		return value.AddDate(0, 1, 0)
	default:
		return value.AddDate(0, 0, 1)
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

package dependency

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	reportv1 "taskmanager/common/gen/go/report/v1"
	"taskmanager/service/report/internal/model"
)

func QueryFromParts(profileID string, from, to *timestamppb.Timestamp, includeArchived bool) model.Query {
	return model.Query{
		ProfileID:       profileID,
		From:            timeFromProto(from),
		To:              timeFromProto(to),
		IncludeArchived: includeArchived,
	}
}

func IntervalFromProto(interval reportv1.ReportInterval) model.Interval {
	switch interval {
	case reportv1.ReportInterval_REPORT_INTERVAL_WEEK:
		return model.IntervalWeek
	case reportv1.ReportInterval_REPORT_INTERVAL_MONTH:
		return model.IntervalMonth
	case reportv1.ReportInterval_REPORT_INTERVAL_DAY:
		return model.IntervalDay
	default:
		return ""
	}
}

func IntervalToProto(interval model.Interval) reportv1.ReportInterval {
	switch interval {
	case model.IntervalWeek:
		return reportv1.ReportInterval_REPORT_INTERVAL_WEEK
	case model.IntervalMonth:
		return reportv1.ReportInterval_REPORT_INTERVAL_MONTH
	case model.IntervalDay:
		return reportv1.ReportInterval_REPORT_INTERVAL_DAY
	default:
		return reportv1.ReportInterval_REPORT_INTERVAL_UNSPECIFIED
	}
}

func OverviewToProto(overview model.Overview) *reportv1.ReportOverview {
	return &reportv1.ReportOverview{
		ProfileId:        overview.ProfileID,
		From:             timestamppb.New(overview.From),
		To:               timestamppb.New(overview.To),
		TotalTasks:       overview.TotalTasks,
		RootTasks:        overview.RootTasks,
		SubtaskCount:     overview.SubtaskCount,
		TodoTasks:        overview.TodoTasks,
		InProgressTasks:  overview.InProgressTasks,
		DoneTasks:        overview.DoneTasks,
		CancelledTasks:   overview.CancelledTasks,
		OverdueTasks:     overview.OverdueTasks,
		DueSoonTasks:     overview.DueSoonTasks,
		CreatedInRange:   overview.CreatedInRange,
		CompletedInRange: overview.CompletedInRange,
		CompletionRate:   overview.CompletionRate,
		OverdueRate:      overview.OverdueRate,
	}
}

func StatusBreakdownToProto(breakdown model.StatusBreakdown) *reportv1.GetStatusBreakdownResponse {
	items := make([]*reportv1.StatusBreakdownItem, 0, len(breakdown.Items))
	for _, item := range breakdown.Items {
		items = append(items, &reportv1.StatusBreakdownItem{
			StatusId:   item.StatusID,
			StatusName: item.StatusName,
			StatusSlug: item.StatusSlug,
			Color:      item.Color,
			Category:   item.Category,
			Count:      item.Count,
			Percentage: item.Percentage,
		})
	}
	return &reportv1.GetStatusBreakdownResponse{Items: items, Total: breakdown.Total}
}

func TimeSeriesToProto(series model.TimeSeries) *reportv1.GetTaskTimeSeriesResponse {
	points := make([]*reportv1.TimeSeriesPoint, 0, len(series.Points))
	for _, point := range series.Points {
		points = append(points, &reportv1.TimeSeriesPoint{
			BucketStart: timestamppb.New(point.BucketStart),
			Bucket:      point.Bucket,
			Created:     point.Created,
			Completed:   point.Completed,
		})
	}
	return &reportv1.GetTaskTimeSeriesResponse{
		Interval:       IntervalToProto(series.Interval),
		Points:         points,
		TotalCreated:   series.TotalCreated,
		TotalCompleted: series.TotalCompleted,
	}
}

func ActivityToProto(report model.ActivityReport) *reportv1.GetActivityReportResponse {
	categories := make([]*reportv1.CategoryCount, 0, len(report.Backlogs.ByCategory))
	for _, item := range report.Backlogs.ByCategory {
		categories = append(categories, &reportv1.CategoryCount{
			Category: item.Category,
			Count:    item.Count,
		})
	}

	return &reportv1.GetActivityReportResponse{
		ProfileId: report.ProfileID,
		From:      timestamppb.New(report.From),
		To:        timestamppb.New(report.To),
		Events: &reportv1.EventStats{
			TotalEvents:     report.Events.TotalEvents,
			PlannedEvents:   report.Events.PlannedEvents,
			ConfirmedEvents: report.Events.ConfirmedEvents,
			CancelledEvents: report.Events.CancelledEvents,
			UpcomingEvents:  report.Events.UpcomingEvents,
			AllDayEvents:    report.Events.AllDayEvents,
		},
		Schedules: &reportv1.ScheduleStats{
			TotalSchedules:    report.Schedules.TotalSchedules,
			AllDaySchedules:   report.Schedules.AllDaySchedules,
			UpcomingSchedules: report.Schedules.UpcomingSchedules,
			TodaySchedules:    report.Schedules.TodaySchedules,
		},
		Backlogs: &reportv1.BacklogStats{
			TotalBacklogs:    report.Backlogs.TotalBacklogs,
			NewBacklogs:      report.Backlogs.NewBacklogs,
			TriagedBacklogs:  report.Backlogs.TriagedBacklogs,
			ArchivedBacklogs: report.Backlogs.ArchivedBacklogs,
			ByCategory:       categories,
		},
	}
}

func timeFromProto(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.AsTime().UTC()
	return &converted
}

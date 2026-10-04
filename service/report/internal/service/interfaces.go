package service

import (
	"context"
	"time"

	"taskmanager/service/report/internal/model"
)

type TaskDataSource interface {
	FetchStatuses(ctx context.Context, profileID string) ([]model.Status, error)
	FetchTasks(ctx context.Context, profileID string, includeArchived bool) ([]model.Task, error)
}

type ActivityDataSource interface {
	FetchEvents(ctx context.Context, profileID string) ([]model.Event, error)
	FetchSchedules(ctx context.Context, profileID string) ([]model.Schedule, error)
	FetchBacklogs(ctx context.Context, profileID string) ([]model.BacklogItem, error)
}

type ReportService interface {
	Overview(ctx context.Context, q model.Query) (model.Overview, error)
	StatusBreakdown(ctx context.Context, q model.Query) (model.StatusBreakdown, error)
	TimeSeries(ctx context.Context, q model.Query, interval model.Interval) (model.TimeSeries, error)
	Activity(ctx context.Context, q model.Query) (model.ActivityReport, error)
}

type Clock func() time.Time

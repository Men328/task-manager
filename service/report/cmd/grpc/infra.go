package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	eventv1 "taskmanager/common/gen/go/event/v1"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/report/internal/config"
	"taskmanager/service/report/internal/repository"
	"taskmanager/service/report/internal/service"
)

func newTaskConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.TaskDialTarget(), "task")
}

func newEventConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.EventDialTarget(), "event")
}

func newCalendarConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.CalendarDialTarget(), "calendar")
}

func newBacklogConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.BacklogDialTarget(), "backlog")
}

func dialGRPC(lc fx.Lifecycle, target string, name string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC %s (%s): %w", name, target, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	slog.Info("report kết nối nguồn dữ liệu", "source", name, "target", target)
	return conn, nil
}

func newTaskServiceClient(conn *grpc.ClientConn) taskv1.TaskServiceClient {
	return taskv1.NewTaskServiceClient(conn)
}

func newTaskStatusServiceClient(conn *grpc.ClientConn) taskv1.TaskStatusServiceClient {
	return taskv1.NewTaskStatusServiceClient(conn)
}

func newEventServiceClient(conn *grpc.ClientConn) eventv1.EventServiceClient {
	return eventv1.NewEventServiceClient(conn)
}

func newCalendarServiceClient(conn *grpc.ClientConn) calendarv1.CalendarServiceClient {
	return calendarv1.NewCalendarServiceClient(conn)
}

func newBacklogServiceClient(conn *grpc.ClientConn) backlogv1.BacklogServiceClient {
	return backlogv1.NewBacklogServiceClient(conn)
}

func newTaskDataSource(
	tasks taskv1.TaskServiceClient,
	statuses taskv1.TaskStatusServiceClient,
	cfg config.Config,
) service.TaskDataSource {
	return repository.NewTaskDataSource(tasks, statuses, cfg.TaskTimeout)
}

func newActivityDataSource(
	events eventv1.EventServiceClient,
	schedules calendarv1.CalendarServiceClient,
	backlogs backlogv1.BacklogServiceClient,
	cfg config.Config,
) service.ActivityDataSource {
	return repository.NewActivityDataSource(events, schedules, backlogs, cfg.ActivityTimeout)
}

func newClock() service.Clock {
	return time.Now
}

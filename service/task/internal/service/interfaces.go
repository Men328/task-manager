package service

import (
	"context"
	"time"

	"taskmanager/service/task/internal/model"
)

type TaskRepository interface {
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Get(ctx context.Context, id string) (model.Task, error)
	List(ctx context.Context, f model.TaskFilter) ([]model.Task, error)
	ListChildren(ctx context.Context, parentTaskID string) ([]model.Task, error)
	Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error)
	Delete(ctx context.Context, id string) error
	ChangeStatus(ctx context.Context, id, statusID string, completedAt *time.Time) (model.Task, error)
	CountByStatus(ctx context.Context, statusID string) (int, error)
	AppendStatusLog(ctx context.Context, log model.StatusLog) error
	ListStatusLogs(ctx context.Context, taskID string) ([]model.StatusLog, error)
}

type StatusRepository interface {
	Create(ctx context.Context, s model.Status) (model.Status, error)
	Get(ctx context.Context, id string) (model.Status, error)
	List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error)
	Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error)
	Delete(ctx context.Context, id string) error
}

type TransitionRepository interface {
	Create(ctx context.Context, t model.Transition) (model.Transition, error)
	Get(ctx context.Context, id string) (model.Transition, error)
	List(ctx context.Context, profileID, fromStatusID string) ([]model.Transition, error)
	Update(ctx context.Context, id string, upd model.TransitionUpdate) (model.Transition, error)
	Delete(ctx context.Context, id string) error

	IsAllowed(ctx context.Context, profileID, fromStatusID, toStatusID string) (bool, error)
}

type TaskService interface {
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Get(ctx context.Context, id string) (model.Task, error)
	List(ctx context.Context, f model.TaskFilter) ([]model.Task, error)
	ListChildren(ctx context.Context, parentTaskID string) ([]model.Task, error)
	Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error)
	Delete(ctx context.Context, id string) error
	ChangeStatus(ctx context.Context, id, statusID, note string) (model.Task, error)
}

type TaskStatusService interface {
	Create(ctx context.Context, s model.Status) (model.Status, error)
	Get(ctx context.Context, id string) (model.Status, error)
	List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error)
	Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error)
	Delete(ctx context.Context, id string) error
}

type StatusTransitionService interface {
	Create(ctx context.Context, t model.Transition) (model.Transition, error)
	Get(ctx context.Context, id string) (model.Transition, error)
	List(ctx context.Context, profileID, fromStatusID string) ([]model.Transition, error)
	Update(ctx context.Context, id string, upd model.TransitionUpdate) (model.Transition, error)
	Delete(ctx context.Context, id string) error
	ValidateStatusTransition(ctx context.Context, profileID, fromStatusID, toStatusID string) (allowed bool, reason string, err error)
}

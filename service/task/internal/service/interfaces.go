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
	// SetProjectForSubtree đổi project của task rootID và mọi hậu duệ của nó.
	SetProjectForSubtree(ctx context.Context, rootID, projectID string) error
	SpawnNextOccurrence(ctx context.Context, prevID, expectedNextID string, next model.Task) (model.Task, error)
	Delete(ctx context.Context, id string) error
	ChangeStatus(ctx context.Context, id, statusID string, completedAt *time.Time) (model.Task, error)
	CountByStatus(ctx context.Context, statusID string) (int, error)
	AppendStatusLog(ctx context.Context, log model.StatusLog) error
	ListStatusLogs(ctx context.Context, taskID string) ([]model.StatusLog, error)
}

// ProjectChecker xác minh project tồn tại, thuộc profile và chưa archive.
type ProjectChecker interface {
	CheckAssignable(ctx context.Context, profileID, projectID string) error
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

type ReminderStore interface {
	ClaimDueReminders(ctx context.Context, limit int) ([]model.Task, error)
	ReleaseReminder(ctx context.Context, id string, claimedAt time.Time) error
}

type NoticeSender interface {
	SendTaskReminder(ctx context.Context, t model.Task) error
}

type TaskService interface {
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Get(ctx context.Context, id string) (model.Task, error)
	List(ctx context.Context, f model.TaskFilter) (model.TaskPage, error)
	ListChildren(ctx context.Context, parentTaskID string) ([]model.Task, error)
	Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error)
	Delete(ctx context.Context, id string) error
	ChangeStatus(ctx context.Context, id, statusID, note string) (model.StatusChange, error)
}

type TaskStatusService interface {
	Create(ctx context.Context, s model.Status) (model.Status, error)
	Get(ctx context.Context, id string) (model.Status, error)
	List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error)
	Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error)
	Delete(ctx context.Context, id string) error
	SeedDefaultStatuses(ctx context.Context, profileID string) ([]model.Status, []model.Transition, error)
}

type StatusTransitionService interface {
	Create(ctx context.Context, t model.Transition) (model.Transition, error)
	Get(ctx context.Context, id string) (model.Transition, error)
	List(ctx context.Context, profileID, fromStatusID string) ([]model.Transition, error)
	Update(ctx context.Context, id string, upd model.TransitionUpdate) (model.Transition, error)
	Delete(ctx context.Context, id string) error
	ValidateStatusTransition(ctx context.Context, profileID, fromStatusID, toStatusID string) (allowed bool, reason string, err error)
}

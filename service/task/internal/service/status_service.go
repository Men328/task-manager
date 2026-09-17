package service

import (
	"context"

	"taskmanager/service/task/internal/model"
)

type taskStatusService struct {
	statuses StatusRepository
	tasks    TaskRepository
}

func NewTaskStatusService(statuses StatusRepository, tasks TaskRepository) TaskStatusService {
	return &taskStatusService{statuses: statuses, tasks: tasks}
}

func (s *taskStatusService) Create(ctx context.Context, st model.Status) (model.Status, error) {
	return s.statuses.Create(ctx, st)
}

func (s *taskStatusService) Get(ctx context.Context, id string) (model.Status, error) {
	return s.statuses.Get(ctx, id)
}

func (s *taskStatusService) List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error) {
	return s.statuses.List(ctx, profileID, includeArchived)
}

func (s *taskStatusService) Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error) {
	return s.statuses.Update(ctx, id, upd)
}

func (s *taskStatusService) Delete(ctx context.Context, id string) error {
	inUse, err := s.tasks.CountByStatus(ctx, id)
	if err != nil {
		return err
	}
	if inUse > 0 {
		return model.NewError(model.ErrorKindStatusInUse,
			"status đang được %d task sử dụng, hãy archive thay vì xoá", inUse)
	}
	return s.statuses.Delete(ctx, id)
}

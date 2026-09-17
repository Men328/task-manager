package service

import (
	"context"
	"time"

	"taskmanager/service/task/internal/model"
)

type taskService struct {
	tasks       TaskRepository
	statuses    StatusRepository
	transitions TransitionRepository
}

func NewTaskService(
	tasks TaskRepository,
	statuses StatusRepository,
	transitions TransitionRepository,
) TaskService {
	return &taskService{tasks: tasks, statuses: statuses, transitions: transitions}
}

func (s *taskService) Create(ctx context.Context, t model.Task) (model.Task, error) {
	statusID, err := s.resolveStatusID(ctx, t.ProfileID, t.StatusID)
	if err != nil {
		return model.Task{}, err
	}
	t.StatusID = statusID

	if parentID := t.ParentTaskID; parentID != "" {
		parent, err := s.tasks.Get(ctx, parentID)
		if err != nil {
			return model.Task{}, err
		}
		if parent.ProfileID != t.ProfileID {
			return model.Task{}, model.NewError(model.ErrorKindParentNotInProfile, "task cha không thuộc profile này")
		}
	}

	created, err := s.tasks.Create(ctx, t)
	if err != nil {
		return model.Task{}, err
	}

	_ = s.tasks.AppendStatusLog(ctx, model.StatusLog{
		TaskID:     created.ID,
		ProfileID:  created.ProfileID,
		ToStatusID: created.StatusID,
		Note:       "tạo task",
		ChangedAt:  time.Now().UTC(),
	})

	return created, nil
}

func (s *taskService) Get(ctx context.Context, id string) (model.Task, error) {
	return s.tasks.Get(ctx, id)
}

func (s *taskService) List(ctx context.Context, f model.TaskFilter) ([]model.Task, error) {
	return s.tasks.List(ctx, f)
}

func (s *taskService) ListChildren(ctx context.Context, parentTaskID string) ([]model.Task, error) {
	return s.tasks.ListChildren(ctx, parentTaskID)
}

func (s *taskService) Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error) {
	return s.tasks.Update(ctx, id, upd)
}

func (s *taskService) Delete(ctx context.Context, id string) error {
	children, err := s.tasks.ListChildren(ctx, id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return model.NewError(model.ErrorKindTaskHasSubtasks,
			"task còn %d task con, hãy xoá/chuyển task con trước", len(children))
	}
	return s.tasks.Delete(ctx, id)
}

func (s *taskService) ChangeStatus(ctx context.Context, id, statusID, note string) (model.Task, error) {
	task, err := s.tasks.Get(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if task.StatusID == statusID {
		return model.Task{}, model.NewError(model.ErrorKindTaskStatusSame, "task đang ở status này rồi")
	}

	target, err := s.statuses.Get(ctx, statusID)
	if err != nil {
		return model.Task{}, err
	}
	if target.ProfileID != task.ProfileID {
		return model.Task{}, model.NewError(model.ErrorKindStatusProfileMismatch, "status không thuộc profile của task")
	}

	allowed, err := s.transitions.IsAllowed(ctx, task.ProfileID, task.StatusID, target.ID)
	if err != nil {
		return model.Task{}, err
	}
	if !allowed {
		return model.Task{}, model.NewError(model.ErrorKindTransitionNotAllowed,
			"không được phép chuyển từ status %s sang %s", task.StatusID, target.ID)
	}

	var completedAt *time.Time
	if target.Category == model.TaskStatusCategoryDone {
		now := time.Now().UTC()
		completedAt = &now
	} else if task.CompletedAt != nil {
		completedAt = nil
	}

	updated, err := s.tasks.ChangeStatus(ctx, task.ID, target.ID, completedAt)
	if err != nil {
		return model.Task{}, err
	}

	_ = s.tasks.AppendStatusLog(ctx, model.StatusLog{
		TaskID:       task.ID,
		ProfileID:    task.ProfileID,
		FromStatusID: task.StatusID,
		ToStatusID:   target.ID,
		Note:         note,
		ChangedAt:    time.Now().UTC(),
	})

	return updated, nil
}

func (s *taskService) resolveStatusID(ctx context.Context, profileID, statusID string) (string, error) {
	if statusID != "" {
		st, err := s.statuses.Get(ctx, statusID)
		if err != nil {
			return "", err
		}
		if st.ProfileID != profileID {
			return "", model.NewError(model.ErrorKindStatusNotInProfile, "status không thuộc profile này")
		}
		return st.ID, nil
	}

	items, err := s.statuses.List(ctx, profileID, false)
	if err != nil {
		return "", err
	}
	for _, st := range items {
		if st.IsDefault {
			return st.ID, nil
		}
	}
	if len(items) > 0 {
		return items[0].ID, nil
	}
	return "", model.NewError(model.ErrorKindNoStatusAvailable, "profile chưa có status nào, hãy tạo status trước")
}

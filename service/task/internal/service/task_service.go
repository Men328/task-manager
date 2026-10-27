package service

import (
	"context"
	"strings"
	"time"

	"taskmanager/service/task/internal/model"
)

type taskService struct {
	tasks       TaskRepository
	statuses    StatusRepository
	transitions TransitionRepository
	projects    ProjectChecker
}

func NewTaskService(
	tasks TaskRepository,
	statuses StatusRepository,
	transitions TransitionRepository,
	projects ProjectChecker,
) TaskService {
	return &taskService{tasks: tasks, statuses: statuses, transitions: transitions, projects: projects}
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
		// Task con luôn nằm cùng project với cha; bỏ qua project client gửi.
		t.ProjectID = parent.ProjectID
	} else {
		if strings.TrimSpace(t.ProjectID) == "" {
			return model.Task{}, model.NewError(model.ErrorKindProjectRequired, "project_id is required")
		}
		if err := s.projects.CheckAssignable(ctx, t.ProfileID, t.ProjectID); err != nil {
			return model.Task{}, err
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

// Update áp các thay đổi thường qua tasks.Update; mọi thay đổi project (đổi
// project của task gốc, hoặc đổi cha sang cây ở project khác) đi qua
// SetProjectForSubtree để cả cây con luôn nằm chung một project.
func (s *taskService) Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error) {
	current, err := s.tasks.Get(ctx, id)
	if err != nil {
		return model.Task{}, err
	}

	if upd.ProjectID != nil && strings.TrimSpace(*upd.ProjectID) == "" {
		return model.Task{}, model.NewError(model.ErrorKindProjectRequired, "project_id must not be empty")
	}

	parentID := current.ParentTaskID
	if upd.ParentTaskID != nil {
		parentID = *upd.ParentTaskID
	}
	parentChanged := parentID != current.ParentTaskID

	targetProject := current.ProjectID
	switch {
	case parentID != "" && (parentChanged || upd.ProjectID != nil):
		parent, err := s.tasks.Get(ctx, parentID)
		if err != nil {
			return model.Task{}, err
		}
		if parent.ProfileID != current.ProfileID {
			return model.Task{}, model.NewError(model.ErrorKindParentNotInProfile, "task cha không thuộc profile này")
		}
		if upd.ProjectID != nil && *upd.ProjectID != parent.ProjectID {
			return model.Task{}, model.NewError(model.ErrorKindSubtaskProjectLocked,
				"task con luôn theo project của task cha; hãy đổi project ở task gốc")
		}
		targetProject = parent.ProjectID
	case parentID == "" && upd.ProjectID != nil && *upd.ProjectID != current.ProjectID:
		if err := s.projects.CheckAssignable(ctx, current.ProfileID, *upd.ProjectID); err != nil {
			return model.Task{}, err
		}
		targetProject = *upd.ProjectID
	}

	// Đổi cha (kèm kiểm tra vòng lặp) và các field khác trước, để lỗi cycle
	// không để lại cây con đã bị đổi project.
	upd.ProjectID = nil
	updated, err := s.tasks.Update(ctx, id, upd)
	if err != nil {
		return model.Task{}, err
	}
	if targetProject == current.ProjectID {
		return updated, nil
	}

	if err := s.tasks.SetProjectForSubtree(ctx, id, targetProject); err != nil {
		return model.Task{}, err
	}
	return s.tasks.Get(ctx, id)
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

package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

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
		if err := s.projects.CheckAssignable(ctx, t.ProfileID, t.ProjectID); err != nil {
			return model.Task{}, err
		}
	} else {
		if strings.TrimSpace(t.ProjectID) == "" {
			return model.Task{}, model.NewError(model.ErrorKindProjectRequired, "project_id is required")
		}
		if err := s.projects.CheckAssignable(ctx, t.ProfileID, t.ProjectID); err != nil {
			return model.Task{}, err
		}
	}

	t.RemindedAt = reminderState(t.RemindAt, time.Now().UTC())

	rule, tz, err := normalizeRecurrence(t.RecurrenceRule, t.RecurrenceTZ, t.DueAt)
	if err != nil {
		return model.Task{}, err
	}
	t.RecurrenceRule, t.RecurrenceTZ = rule, tz
	if rule != "" {
		if t.ID == "" {
			t.ID = uuid.NewString()
		}
		if t.RecurrenceSeriesID == "" {
			t.RecurrenceSeriesID = t.ID
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

func (s *taskService) List(ctx context.Context, f model.TaskFilter) (model.TaskPage, error) {
	if f.OpenOnly {
		closed, err := s.closedStatusIDs(ctx, f.ProfileID)
		if err != nil {
			return model.TaskPage{}, err
		}
		f.ExcludeStatusIDs = append(f.ExcludeStatusIDs, closed...)
	}

	pageSize := f.Limit
	if pageSize > 0 {
		f.Limit = pageSize + 1
	}
	items, err := s.tasks.List(ctx, f)
	if err != nil {
		return model.TaskPage{}, err
	}

	page := model.TaskPage{Tasks: items}
	if pageSize > 0 && len(items) > pageSize {
		page.Tasks = items[:pageSize]
		page.NextOffset = f.Offset + pageSize
	}
	return page, nil
}

func (s *taskService) closedStatusIDs(ctx context.Context, profileID string) ([]string, error) {
	statuses, err := s.statuses.List(ctx, profileID, true)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(statuses))
	for _, st := range statuses {
		if st.Category == model.TaskStatusCategoryDone || st.Category == model.TaskStatusCategoryCancelled {
			out = append(out, st.ID)
		}
	}
	return out, nil
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
	if upd.ClearRemindAt {
		upd.RemindAt = nil
	}
	if upd.ChangesReminder() {
		upd.RemindedAt = reminderState(upd.RemindAt, time.Now().UTC())
	}
	if err := applyRecurrenceUpdate(current, &upd); err != nil {
		return model.Task{}, err
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
		if targetProject != current.ProjectID {
			if err := s.projects.CheckAssignable(ctx, current.ProfileID, targetProject); err != nil {
				return model.Task{}, err
			}
		}
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

func (s *taskService) ChangeStatus(ctx context.Context, id, statusID, note string) (model.StatusChange, error) {
	task, err := s.tasks.Get(ctx, id)
	if err != nil {
		return model.StatusChange{}, err
	}
	if task.StatusID == statusID {
		return model.StatusChange{}, model.NewError(model.ErrorKindTaskStatusSame, "task đang ở status này rồi")
	}

	target, err := s.statuses.Get(ctx, statusID)
	if err != nil {
		return model.StatusChange{}, err
	}
	if target.ProfileID != task.ProfileID {
		return model.StatusChange{}, model.NewError(model.ErrorKindStatusProfileMismatch, "status không thuộc profile của task")
	}

	allowed, err := s.transitions.IsAllowed(ctx, task.ProfileID, task.StatusID, target.ID)
	if err != nil {
		return model.StatusChange{}, err
	}
	if !allowed {
		return model.StatusChange{}, model.NewError(model.ErrorKindTransitionNotAllowed,
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
		return model.StatusChange{}, err
	}

	_ = s.tasks.AppendStatusLog(ctx, model.StatusLog{
		TaskID:       task.ID,
		ProfileID:    task.ProfileID,
		FromStatusID: task.StatusID,
		ToStatusID:   target.ID,
		Note:         note,
		ChangedAt:    time.Now().UTC(),
	})

	result := model.StatusChange{Task: updated}
	if target.Category == model.TaskStatusCategoryDone && updated.RecurrenceRule != "" && !s.nextOccurrenceExists(ctx, updated) {
		next, spawned, err := s.spawnNextOccurrence(ctx, updated)
		switch {
		case err != nil:
			slog.Warn("không tạo được lần lặp kế tiếp", "task_id", updated.ID, "error", err)
			result.NextErr = err
		case spawned:
			result.Next = &next
		}
	}
	return result, nil
}

func applyRecurrenceUpdate(current model.Task, upd *model.TaskUpdate) error {
	if upd.RecurrenceRule == nil {
		if upd.DueAt == nil && upd.RecurrenceTZ == nil {
			return nil
		}
		if current.RecurrenceRule == "" {
			upd.RecurrenceTZ = nil
			return nil
		}
		rule := current.RecurrenceRule
		upd.RecurrenceRule = &rule
	}
	due := current.DueAt
	if upd.DueAt != nil {
		due = upd.DueAt
	}
	tz := current.RecurrenceTZ
	if upd.RecurrenceTZ != nil {
		tz = *upd.RecurrenceTZ
	}
	rule, tz, err := normalizeRecurrence(*upd.RecurrenceRule, tz, due)
	if err != nil {
		return err
	}
	upd.RecurrenceRule = &rule
	upd.RecurrenceTZ = &tz
	if rule != "" && current.RecurrenceSeriesID == "" {
		series := current.ID
		upd.RecurrenceSeriesID = &series
	}
	return nil
}

func (s *taskService) nextOccurrenceExists(ctx context.Context, t model.Task) bool {
	if t.RecurrenceNextID == "" {
		return false
	}
	_, err := s.tasks.Get(ctx, t.RecurrenceNextID)
	return !errors.Is(err, model.ErrNotFound)
}

func (s *taskService) spawnNextOccurrence(ctx context.Context, done model.Task) (model.Task, bool, error) {
	if done.DueAt == nil {
		return model.Task{}, false, nil
	}
	now := time.Now().UTC()
	occ, ok, err := nextOccurrence(done.RecurrenceRule, done.RecurrenceTZ, *done.DueAt, now)
	if err != nil || !ok {
		return model.Task{}, false, err
	}

	shift := occ.due.Sub(*done.DueAt)
	moved := func(at *time.Time) *time.Time {
		if at == nil {
			return nil
		}
		v := at.Add(shift)
		return &v
	}

	statusID, err := s.resolveStatusID(ctx, done.ProfileID, "")
	if err != nil {
		return model.Task{}, false, err
	}
	if err := s.projects.CheckAssignable(ctx, done.ProfileID, done.ProjectID); err != nil {
		return model.Task{}, false, err
	}

	due := occ.due
	next := model.Task{
		ID:                 uuid.NewString(),
		ProfileID:          done.ProfileID,
		ProjectID:          done.ProjectID,
		ParentTaskID:       done.ParentTaskID,
		StatusID:           statusID,
		Title:              done.Title,
		Description:        done.Description,
		Priority:           done.Priority,
		StartAt:            moved(done.StartAt),
		DueAt:              &due,
		RemindAt:           moved(done.RemindAt),
		RecurrenceRule:     occ.rule,
		RecurrenceTZ:       done.RecurrenceTZ,
		RecurrenceSeriesID: done.RecurrenceSeriesID,
	}
	next.RemindedAt = reminderState(next.RemindAt, now)

	created, err := s.tasks.SpawnNextOccurrence(ctx, done.ID, done.RecurrenceNextID, next)
	if errors.Is(err, model.ErrAlreadyExists) {
		return model.Task{}, false, nil
	}
	if err != nil {
		return model.Task{}, false, err
	}
	_ = s.tasks.AppendStatusLog(ctx, model.StatusLog{
		TaskID:     created.ID,
		ProfileID:  created.ProfileID,
		ToStatusID: created.StatusID,
		Note:       "lặp lại",
		ChangedAt:  now,
	})
	return created, true, nil
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

func reminderState(remindAt *time.Time, now time.Time) *time.Time {
	if remindAt == nil || remindAt.After(now) {
		return nil
	}
	return &now
}

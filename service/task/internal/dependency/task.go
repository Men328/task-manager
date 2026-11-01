package dependency

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

func ValidateCreateTask(req *taskv1.CreateTaskRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.TaskProfileIDRequired)
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return errorcode.Error(errorcode.TaskTitleRequired)
	}
	return nil
}

func ValidateTaskID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.TaskIDRequired)
	}
	return nil
}

func ValidateChangeTaskStatus(req *taskv1.ChangeTaskStatusRequest) error {
	if req.GetId() == "" || req.GetStatusId() == "" {
		return errorcode.Error(errorcode.TaskIDAndStatusIDRequired)
	}
	return nil
}

func TaskFromCreateRequest(req *taskv1.CreateTaskRequest) model.Task {
	return model.Task{
		ProfileID:    req.GetProfileId(),
		ProjectID:    req.GetProjectId(),
		ParentTaskID: req.GetParentTaskId(),
		StatusID:     req.GetStatusId(),
		Title:        req.GetTitle(),
		Description:  req.GetDescription(),
		Priority:     taskPriorityFromProto(req.GetPriority()),
		Position:     req.GetPosition(),
		StartAt:      timePtr(req.GetStartAt()),
		DueAt:        timePtr(req.GetDueAt()),
	}
}

const maxPageSize = 500

const pageTokenPrefix = "o:"

const (
	minSearchQueryRunes = 2
	maxSearchQueryRunes = 200
)

func normalizeSearchQuery(raw string) (string, error) {
	query := strings.TrimSpace(raw)
	if query == "" {
		return "", nil
	}
	runes := []rune(query)
	if len(runes) < minSearchQueryRunes {
		return "", errorcode.Error(errorcode.TaskSearchQueryTooShort)
	}
	if len(runes) > maxSearchQueryRunes {
		query = strings.TrimSpace(string(runes[:maxSearchQueryRunes]))
	}
	return query, nil
}

func usesListExtensions(req *taskv1.ListTasksRequest) bool {
	return req.GetDueFrom() != nil || req.GetDueTo() != nil || req.HasDueAt != nil ||
		req.GetOpenOnly() || len(req.GetPriorities()) > 0 ||
		req.GetSortField() != taskv1.TaskSortField_TASK_SORT_FIELD_UNSPECIFIED || req.GetSortDesc() ||
		req.GetPageSize() != 0 || req.GetPageToken() != "" || req.GetQuery() != ""
}

func TaskFilterFromRequest(req *taskv1.ListTasksRequest) (model.TaskFilter, error) {
	f := model.TaskFilter{
		ProfileID:       req.GetProfileId(),
		ProjectID:       req.GetProjectId(),
		ParentTaskID:    req.GetParentTaskId(),
		RootOnly:        req.GetRootOnly(),
		StatusID:        req.GetStatusId(),
		IncludeArchived: req.GetIncludeArchived(),
	}
	if !usesListExtensions(req) {
		return f, nil
	}
	if req.GetProfileId() == "" {
		return model.TaskFilter{}, errorcode.Error(errorcode.TaskProfileIDRequired)
	}

	f.DueFrom = timePtr(req.GetDueFrom())
	f.DueTo = timePtr(req.GetDueTo())
	f.HasDueAt = req.HasDueAt
	hasRange := f.DueFrom != nil || f.DueTo != nil
	if f.DueFrom != nil && f.DueTo != nil && !f.DueFrom.Before(*f.DueTo) {
		return model.TaskFilter{}, errorcode.Error(errorcode.TaskDueRangeInvalid)
	}
	if hasRange && f.HasDueAt != nil && !*f.HasDueAt {
		return model.TaskFilter{}, errorcode.Error(errorcode.TaskDueRangeInvalid)
	}

	f.OpenOnly = req.GetOpenOnly()
	for _, p := range req.GetPriorities() {
		f.Priorities = append(f.Priorities, taskPriorityFromProto(p))
	}
	f.SortField = model.TaskSortField(req.GetSortField())
	f.SortDesc = req.GetSortDesc()

	size := int(req.GetPageSize())
	if size < 0 {
		return model.TaskFilter{}, errorcode.Error(errorcode.TaskPageSizeInvalid)
	}
	f.Limit = min(size, maxPageSize)

	query, err := normalizeSearchQuery(req.GetQuery())
	if err != nil {
		return model.TaskFilter{}, err
	}
	f.Query = query

	if token := req.GetPageToken(); token != "" {
		offset, ok := decodePageToken(token)
		if !ok {
			return model.TaskFilter{}, errorcode.Error(errorcode.TaskPageTokenInvalid)
		}
		f.Offset = offset
	}
	return f, nil
}

func EncodePageToken(offset int) string {
	if offset <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(pageTokenPrefix + strconv.Itoa(offset)))
}

func decodePageToken(token string) (int, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, false
	}
	value, found := strings.CutPrefix(string(raw), pageTokenPrefix)
	if !found {
		return 0, false
	}
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 {
		return 0, false
	}
	return offset, true
}

func TaskUpdateFromRequest(req *taskv1.UpdateTaskRequest) model.TaskUpdate {
	return model.TaskUpdate{
		ProjectID:    req.ProjectId,
		ParentTaskID: req.ParentTaskId,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     taskPriorityPtr(req.Priority),
		Position:     req.Position,
		StartAt:      timePtr(req.GetStartAt()),
		DueAt:        timePtr(req.GetDueAt()),
		IsArchived:   req.IsArchived,
	}
}

func TaskToProto(t model.Task) *taskv1.Task {
	return &taskv1.Task{
		Id:           t.ID,
		ProfileId:    t.ProfileID,
		ProjectId:    t.ProjectID,
		ParentTaskId: t.ParentTaskID,
		StatusId:     t.StatusID,
		Title:        t.Title,
		Description:  t.Description,
		Priority:     taskPriorityToProto(t.Priority),
		Position:     t.Position,
		StartAt:      timestamp(t.StartAt),
		DueAt:        timestamp(t.DueAt),
		CompletedAt:  timestamp(t.CompletedAt),
		IsArchived:   t.IsArchived,
		CreatedAt:    timestamppb.New(t.CreatedAt),
		UpdatedAt:    timestamppb.New(t.UpdatedAt),
	}
}

func taskPriorityFromProto(p taskv1.TaskPriority) model.TaskPriority {
	return model.TaskPriority(p)
}

func taskPriorityToProto(p model.TaskPriority) taskv1.TaskPriority {
	return taskv1.TaskPriority(p)
}

func taskPriorityPtr(p *taskv1.TaskPriority) *model.TaskPriority {
	if p == nil {
		return nil
	}
	v := taskPriorityFromProto(*p)
	return &v
}

func timestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().UTC()
	return &t
}

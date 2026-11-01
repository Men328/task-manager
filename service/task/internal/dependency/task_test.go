package dependency

import (
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

func wantErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s, got nil", code)
	}
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("grpc code = %v, want InvalidArgument (%v)", got, err)
	}
	if want := errorcode.Error(code); err.Error() != want.Error() {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestTaskFilterLegacyRequestKeepsOldBehaviour(t *testing.T) {
	f, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{RootOnly: true})
	if err != nil {
		t.Fatalf("legacy request without profile_id must still work: %v", err)
	}
	if !f.RootOnly || f.Limit != 0 {
		t.Fatalf("filter = %+v", f)
	}
}

func TestTaskFilterExtensionsRequireProfile(t *testing.T) {
	_, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{OpenOnly: true})
	wantErrorCode(t, err, errorcode.TaskProfileIDRequired)
}

func TestTaskFilterDueRangeValidation(t *testing.T) {
	day := timestamppb.New(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	no := false

	_, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", DueFrom: day, DueTo: day})
	wantErrorCode(t, err, errorcode.TaskDueRangeInvalid)

	_, err = TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", DueTo: day, HasDueAt: &no})
	wantErrorCode(t, err, errorcode.TaskDueRangeInvalid)
}

func TestTaskFilterPageSize(t *testing.T) {
	_, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", PageSize: -1})
	wantErrorCode(t, err, errorcode.TaskPageSizeInvalid)

	f, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", PageSize: 10_000})
	if err != nil || f.Limit != maxPageSize {
		t.Fatalf("page size must be clamped to %d, got %d (%v)", maxPageSize, f.Limit, err)
	}
}

func TestTaskFilterMapsSortAndPriorities(t *testing.T) {
	f, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{
		ProfileId:  "u1",
		Priorities: []taskv1.TaskPriority{taskv1.TaskPriority_TASK_PRIORITY_HIGH},
		SortField:  taskv1.TaskSortField_TASK_SORT_FIELD_PRIORITY,
		SortDesc:   true,
	})
	if err != nil {
		t.Fatalf("TaskFilterFromRequest: %v", err)
	}
	if f.SortField != model.TaskSortPriority || !f.SortDesc || len(f.Priorities) != 1 || f.Priorities[0] != model.TaskPriorityHigh {
		t.Fatalf("filter = %+v", f)
	}
}

func TestPageTokenRoundTrip(t *testing.T) {
	if EncodePageToken(0) != "" {
		t.Fatal("offset 0 must encode to empty token")
	}
	token := EncodePageToken(40)
	f, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", PageSize: 20, PageToken: token})
	if err != nil || f.Offset != 40 {
		t.Fatalf("offset = %d, err = %v; want 40", f.Offset, err)
	}

	for _, bad := range []string{"%%%", "eDox", EncodePageToken(1)[:2]} {
		_, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", PageToken: bad})
		wantErrorCode(t, err, errorcode.TaskPageTokenInvalid)
	}
}

func TestTaskFilterSearchQuery(t *testing.T) {
	_, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{Query: "bao cao"})
	wantErrorCode(t, err, errorcode.TaskProfileIDRequired)

	_, err = TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", Query: "  đ "})
	wantErrorCode(t, err, errorcode.TaskSearchQueryTooShort)

	f, err := TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", Query: "  Báo cáo  "})
	if err != nil || f.Query != "Báo cáo" {
		t.Fatalf("query = %q, err = %v; want trimmed", f.Query, err)
	}

	f, _ = TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", Query: strings.Repeat("ữ", 250)})
	if got := len([]rune(f.Query)); got != maxSearchQueryRunes {
		t.Fatalf("long query has %d runes, want cut to %d", got, maxSearchQueryRunes)
	}

	f, err = TaskFilterFromRequest(&taskv1.ListTasksRequest{ProfileId: "u1", Query: "   "})
	if err != nil || f.Query != "" {
		t.Fatalf("blank query must be ignored, got %q, %v", f.Query, err)
	}
}

func TestValidateUpdateTaskReminderConflict(t *testing.T) {
	at := timestamppb.New(time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC))
	err := ValidateUpdateTask(&taskv1.UpdateTaskRequest{Id: "t1", RemindAt: at, ClearRemindAt: true})
	wantErrorCode(t, err, errorcode.TaskRemindAtConflict)

	if err := ValidateUpdateTask(&taskv1.UpdateTaskRequest{Id: "t1", RemindAt: at}); err != nil {
		t.Fatalf("setting a reminder must be valid: %v", err)
	}
	upd := TaskUpdateFromRequest(&taskv1.UpdateTaskRequest{Id: "t1", ClearRemindAt: true})
	if !upd.ClearRemindAt || !upd.ChangesReminder() {
		t.Fatalf("clear flag not mapped: %+v", upd)
	}
}

func TestErrorCodeMatchesGRPCError(t *testing.T) {
	for _, err := range []error{
		model.ErrNotFound, model.ErrAlreadyExists, model.ErrInUse, model.ErrCycle,
		model.NewError(model.ErrorKindRecurrenceInvalid, "x"), errors.New("boom"),
	} {
		if got, want := ToGRPCError(err).Error(), errorcode.Error(ErrorCode(err)).Error(); got != want {
			t.Fatalf("%v: grpc %q, code %q", err, got, want)
		}
	}
	if ErrorCode(model.ErrCycle) != errorcode.TaskCycleDetected {
		t.Fatal("ErrCycle must map to TASK_CYCLE_DETECTED")
	}
}

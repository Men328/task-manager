package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/task/internal/model"
)

func ToGRPCError(err error) error {
	return errorcode.Error(ErrorCode(err))
}

func ErrorCode(err error) string {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return codeForKind(domainErr.Kind)
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.CommonNotFound
	case errors.Is(err, model.ErrAlreadyExists):
		return errorcode.TaskTransitionAlreadyExists
	case errors.Is(err, model.ErrInUse):
		return errorcode.CommonFailedPrecondition
	case errors.Is(err, model.ErrCycle):
		return errorcode.TaskCycleDetected
	default:
		return errorcode.CommonInternal
	}
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindStatusNotInProfile:
		return errorcode.TaskStatusNotInProfile
	case model.ErrorKindParentNotInProfile:
		return errorcode.TaskParentNotInProfile
	case model.ErrorKindNoStatusAvailable:
		return errorcode.TaskNoStatusAvailable
	case model.ErrorKindStatusProfileMismatch:
		return errorcode.TaskStatusProfileMismatch
	case model.ErrorKindTaskStatusSame:
		return errorcode.TaskStatusSame
	case model.ErrorKindTransitionNotAllowed:
		return errorcode.TaskTransitionNotAllowed
	case model.ErrorKindTaskHasSubtasks:
		return errorcode.TaskHasSubtasks
	case model.ErrorKindStatusInUse:
		return errorcode.TaskStatusInUse
	case model.ErrorKindStatusSlugAlreadyExists:
		return errorcode.TaskStatusSlugAlreadyExists
	case model.ErrorKindProjectRequired:
		return errorcode.ProjectIDRequired
	case model.ErrorKindProjectNotFound:
		return errorcode.ProjectNotFound
	case model.ErrorKindProjectArchived:
		return errorcode.ProjectArchived
	case model.ErrorKindSubtaskProjectLocked:
		return errorcode.TaskSubtaskProjectLocked
	case model.ErrorKindRecurrenceInvalid:
		return errorcode.TaskRecurrenceInvalid
	default:
		return errorcode.CommonInternal
	}
}

package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/task/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.CommonNotFound)
	case errors.Is(err, model.ErrAlreadyExists):
		return errorcode.Error(errorcode.TaskTransitionAlreadyExists)
	case errors.Is(err, model.ErrInUse):
		return errorcode.Error(errorcode.CommonFailedPrecondition)
	case errors.Is(err, model.ErrCycle):
		return errorcode.Error(errorcode.TaskCycleDetected)
	default:
		return errorcode.Error(errorcode.CommonInternal)
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
	default:
		return errorcode.CommonInternal
	}
}

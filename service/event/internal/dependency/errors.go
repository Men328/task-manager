package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/event/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.EventNotFound)
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.CommonInvalidArgument)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindTimeRangeInvalid:
		return errorcode.EventTimeRangeInvalid
	case model.ErrorKindStatusInvalid:
		return errorcode.CommonInvalidArgument
	case model.ErrorKindProjectRequired:
		return errorcode.ProjectIDRequired
	case model.ErrorKindProjectNotFound:
		return errorcode.ProjectNotFound
	case model.ErrorKindProjectArchived:
		return errorcode.ProjectArchived
	default:
		return errorcode.CommonInternal
	}
}

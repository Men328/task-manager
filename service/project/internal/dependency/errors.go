package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/project/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.ProjectNotFound)
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.ProjectNameRequired)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindNameDuplicate:
		return errorcode.ProjectNameDuplicate
	case model.ErrorKindArchived:
		return errorcode.ProjectArchived
	case model.ErrorKindDefaultNotArchivable:
		return errorcode.ProjectDefaultNotArchivable
	case model.ErrorKindNameTooLong:
		return errorcode.ProjectNameTooLong
	case model.ErrorKindColorInvalid:
		return errorcode.ProjectColorInvalid
	default:
		return errorcode.CommonInternal
	}
}

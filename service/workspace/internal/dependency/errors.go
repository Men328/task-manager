package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/workspace/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind), domainErr.Message)
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.WorkspaceNotFound, err.Error())
	case errors.Is(err, model.ErrAlreadyExists):
		return errorcode.Error(errorcode.WorkspaceSlugAlreadyExists, err.Error())
	default:
		return errorcode.Error(errorcode.CommonInternal, err.Error())
	}
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindSlugAlreadyExists:
		return errorcode.WorkspaceSlugAlreadyExists
	default:
		return errorcode.CommonInternal
	}
}

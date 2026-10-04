package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/attachment/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.AttachmentNotFound)
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.CommonInvalidArgument)
	case errors.Is(err, model.ErrTooLarge):
		return errorcode.Error(errorcode.AttachmentFileTooLarge)
	case errors.Is(err, model.ErrStorageFailed):
		return errorcode.Error(errorcode.AttachmentStorageFailed)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindOwnerTypeInvalid:
		return errorcode.AttachmentOwnerTypeInvalid
	case model.ErrorKindFileTooLarge:
		return errorcode.AttachmentFileTooLarge
	case model.ErrorKindFileRequired:
		return errorcode.AttachmentFileRequired
	case model.ErrorKindStorageFailed:
		return errorcode.AttachmentStorageFailed
	default:
		return errorcode.CommonInternal
	}
}

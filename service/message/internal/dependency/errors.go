package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/message/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}
	if errors.Is(err, model.ErrUnavailable) {
		return errorcode.Error(errorcode.CommonUnavailable)
	}
	return errorcode.Error(errorcode.CommonInternal)
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindInvalidArgument:
		return errorcode.CommonInvalidArgument
	case model.ErrorKindBodyRequired:
		return errorcode.MessageBodyRequired
	case model.ErrorKindBodyTooLong:
		return errorcode.MessageBodyTooLong
	case model.ErrorKindCannotMessageSelf:
		return errorcode.MessageCannotMessageSelf
	case model.ErrorKindNotFriends:
		return errorcode.MessageNotFriends
	default:
		return errorcode.CommonInternal
	}
}

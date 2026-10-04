package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/notification/internal/model"
)

func ToGRPCError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.NotificationNotFound)
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.CommonInvalidArgument)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

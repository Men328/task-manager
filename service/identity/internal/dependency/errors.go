package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/identity/internal/model"
)

func ToGRPCError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.IdentityProfileNotFound)
	case errors.Is(err, model.ErrEmailExists):
		return errorcode.Error(errorcode.IdentityEmailAlreadyExists)
	case errors.Is(err, model.ErrProfileInactive):
		return errorcode.Error(errorcode.IdentityAuthProfileInactive)
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.CommonInvalidArgument)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

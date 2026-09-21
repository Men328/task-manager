package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/identity/internal/model"
)

func ToGRPCError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.IdentityProfileNotFound, err.Error())
	case errors.Is(err, model.ErrEmailExists):
		return errorcode.Error(errorcode.IdentityEmailAlreadyExists, err.Error())
	case errors.Is(err, model.ErrProfileInactive):
		return errorcode.Error(errorcode.IdentityAuthProfileInactive, err.Error())
	default:
		return errorcode.Error(errorcode.CommonInternal, err.Error())
	}
}

package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/identity/internal/model"
)

func ToGRPCError(err error) error {
	if errors.Is(err, model.ErrNotFound) {
		return errorcode.Error(errorcode.IdentityProfileNotFound, err.Error())
	}
	return errorcode.Error(errorcode.CommonInternal, err.Error())
}

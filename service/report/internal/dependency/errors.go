package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/report/internal/model"
)

func ValidateProfileID(profileID string) error {
	if profileID == "" {
		return errorcode.Error(errorcode.ReportProfileIDRequired)
	}
	return nil
}

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		switch domainErr.Kind {
		case model.ErrorKindRangeInvalid:
			return errorcode.Error(errorcode.ReportRangeInvalid)
		case model.ErrorKindIntervalInvalid:
			return errorcode.Error(errorcode.ReportIntervalInvalid)
		case model.ErrorKindSourceFailed:
			return errorcode.Error(errorcode.ReportTaskSourceFailed)
		default:
			return errorcode.Error(errorcode.CommonInternal)
		}
	}

	switch {
	case errors.Is(err, model.ErrInvalid):
		return errorcode.Error(errorcode.CommonInvalidArgument)
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.CommonNotFound)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

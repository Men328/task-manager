package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/mail-provider/internal/model"
)

func ToGRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, model.ErrNotFound):
		return errorcode.Error(errorcode.MailSubscriptionNotFound)
	case errors.Is(err, model.ErrNotConfigured):
		return errorcode.Error(errorcode.MailNotConfigured)
	case errors.Is(err, model.ErrGmailFailed):
		return errorcode.Error(errorcode.MailGmailFailed)
	case errors.Is(err, model.ErrHistoryGone):
		return errorcode.Error(errorcode.MailGmailFailed)
	case errors.Is(err, model.ErrTokenRefresh):
		return errorcode.Error(errorcode.MailTokenRefreshFailed)
	case errors.Is(err, model.ErrAnalyzeFailed):
		return errorcode.Error(errorcode.MailAnalyzeFailed)
	case errors.Is(err, model.ErrTaskCreate):
		return errorcode.Error(errorcode.MailTaskCreateFailed)
	case errors.Is(err, model.ErrNoWorkspace):
		return errorcode.Error(errorcode.MailNoWorkspace)
	case errors.Is(err, model.ErrQueueFull):
		return errorcode.Error(errorcode.CommonUnavailable)
	case errors.Is(err, model.ErrUnauthorized):
		return errorcode.Error(errorcode.MailUnauthorized)
	case errors.Is(err, model.ErrInvalidPayload):
		return errorcode.Error(errorcode.MailNotificationInvalid)
	default:
		return errorcode.Error(errorcode.CommonInternal)
	}
}

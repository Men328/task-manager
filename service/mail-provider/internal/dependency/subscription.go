package dependency

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/mail-provider/internal/model"
)

func ValidateSubscribe(req *mailv1.SubscribeRequest) error {
	if strings.TrimSpace(req.GetProfileId()) == "" {
		return errorcode.Error(errorcode.MailProfileIDRequired)
	}
	if strings.TrimSpace(req.GetEmail()) == "" {
		return errorcode.Error(errorcode.MailEmailRequired)
	}
	if strings.TrimSpace(req.GetAccessToken()) == "" {
		return errorcode.Error(errorcode.MailAccessTokenRequired)
	}
	return nil
}

func ValidateProfileID(profileID string) error {
	if strings.TrimSpace(profileID) == "" {
		return errorcode.Error(errorcode.MailProfileIDRequired)
	}
	return nil
}

func ValidateNotification(email string, historyID string) error {
	if strings.TrimSpace(email) == "" || strings.TrimSpace(historyID) == "" {
		return errorcode.Error(errorcode.MailNotificationInvalid)
	}
	return nil
}

func SubscribeInputFromRequest(req *mailv1.SubscribeRequest) model.SubscribeInput {
	return model.SubscribeInput{
		ProfileID:            strings.TrimSpace(req.GetProfileId()),
		Email:                strings.TrimSpace(req.GetEmail()),
		AccessToken:          strings.TrimSpace(req.GetAccessToken()),
		RefreshToken:         strings.TrimSpace(req.GetRefreshToken()),
		AccessTokenExpiresAt: timestampFromProto(req.GetAccessTokenExpiresAt()),
	}
}

func SubscriptionToProto(sub model.Subscription) *mailv1.Subscription {
	out := &mailv1.Subscription{
		ProfileId: sub.ProfileID,
		Email:     sub.Email,
		HistoryId: sub.HistoryID,
	}
	if !sub.WatchExpiresAt.IsZero() {
		out.WatchExpiresAt = timestamppb.New(sub.WatchExpiresAt)
	}
	if !sub.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(sub.CreatedAt)
	}
	if !sub.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(sub.UpdatedAt)
	}
	return out
}

func timestampFromProto(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime().UTC()
}

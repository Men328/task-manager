package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/notification/internal/model"
)

func ValidateCreateNotice(req *notificationv1.CreateNoticeRequest) error {
	if strings.TrimSpace(req.GetProfileId()) == "" {
		return errorcode.Error(errorcode.NotificationProfileIDRequired)
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return errorcode.Error(errorcode.NotificationTitleRequired)
	}
	if targetTypeFromProto(req.GetTargetType()) == "" {
		return errorcode.Error(errorcode.NotificationTargetTypeInvalid)
	}
	if strings.TrimSpace(req.GetTargetId()) == "" {
		return errorcode.Error(errorcode.NotificationTargetIDRequired)
	}
	return nil
}

func NoticeFromCreateRequest(req *notificationv1.CreateNoticeRequest) model.Notice {
	return model.Notice{
		ProfileID:  req.GetProfileId(),
		Type:       req.GetType(),
		Title:      req.GetTitle(),
		Body:       req.GetBody(),
		TargetType: targetTypeFromProto(req.GetTargetType()),
		TargetID:   req.GetTargetId(),
		Source:     req.GetSource(),
	}
}

func FilterFromRequest(req *notificationv1.ListNoticesRequest) model.Filter {
	return model.Filter{
		ProfileID:  req.GetProfileId(),
		Limit:      int(req.GetLimit()),
		UnreadOnly: req.GetUnreadOnly(),
	}
}

func NoticeToProto(notice model.Notice) *notificationv1.Notice {
	converted := &notificationv1.Notice{
		Id:         notice.ID,
		ProfileId:  notice.ProfileID,
		Type:       notice.Type,
		Title:      notice.Title,
		Body:       notice.Body,
		TargetType: targetTypeToProto(notice.TargetType),
		TargetId:   notice.TargetID,
		IsRead:     notice.IsRead,
		Source:     notice.Source,
		CreatedAt:  timestamppb.New(notice.CreatedAt),
	}
	if notice.ReadAt != nil {
		converted.ReadAt = timestamppb.New(*notice.ReadAt)
	}
	return converted
}

func targetTypeToProto(targetType string) notificationv1.NoticeTargetType {
	switch targetType {
	case model.TargetTypeTask:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_TASK
	case model.TargetTypeSchedule:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_SCHEDULE
	case model.TargetTypeEvent:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_EVENT
	case model.TargetTypeBacklog:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_BACKLOG
	default:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_UNSPECIFIED
	}
}

func targetTypeFromProto(targetType notificationv1.NoticeTargetType) string {
	switch targetType {
	case notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_TASK:
		return model.TargetTypeTask
	case notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_SCHEDULE:
		return model.TargetTypeSchedule
	case notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_EVENT:
		return model.TargetTypeEvent
	case notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_BACKLOG:
		return model.TargetTypeBacklog
	default:
		return ""
	}
}

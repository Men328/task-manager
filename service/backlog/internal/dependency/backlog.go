package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	"taskmanager/service/backlog/internal/model"
)

func ValidateCreateBacklog(req *backlogv1.CreateBacklogRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.BacklogProfileIDRequired)
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return errorcode.Error(errorcode.BacklogTitleRequired)
	}
	return nil
}

func ValidateBacklogID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.BacklogIDRequired)
	}
	return nil
}

func BacklogFromCreateRequest(req *backlogv1.CreateBacklogRequest) model.Item {
	return model.Item{
		ProfileID:   req.GetProfileId(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Sender:      req.GetSender(),
		Source:      req.GetSource(),
		Category:    req.GetCategory(),
		Reason:      req.GetReason(),
		ObjectKey:   req.GetObjectKey(),
		Status:      statusFromProto(req.GetStatus()),
	}
}

func BacklogFilterFromRequest(req *backlogv1.ListBacklogsRequest) model.Filter {
	return model.Filter{
		ProfileID: req.GetProfileId(),
		Status:    statusFromProto(req.GetStatus()),
		Category:  req.GetCategory(),
	}
}

func BacklogUpdateFromRequest(req *backlogv1.UpdateBacklogRequest) model.Update {
	return model.Update{
		Title:       req.Title,
		Description: req.Description,
		Reason:      req.Reason,
		Status:      updateStatusFromProto(req.Status),
	}
}

func BacklogToProto(item model.Item) *backlogv1.Backlog {
	return &backlogv1.Backlog{
		Id:          item.ID,
		ProfileId:   item.ProfileID,
		Title:       item.Title,
		Description: item.Description,
		Sender:      item.Sender,
		Source:      item.Source,
		Category:    item.Category,
		Reason:      item.Reason,
		ObjectKey:   item.ObjectKey,
		Status:      statusToProto(item.Status),
		CreatedAt:   timestamppb.New(item.CreatedAt),
		UpdatedAt:   timestamppb.New(item.UpdatedAt),
	}
}

func statusToProto(status model.Status) backlogv1.BacklogStatus {
	switch status {
	case model.StatusNew:
		return backlogv1.BacklogStatus_BACKLOG_STATUS_NEW
	case model.StatusTriaged:
		return backlogv1.BacklogStatus_BACKLOG_STATUS_TRIAGED
	case model.StatusArchived:
		return backlogv1.BacklogStatus_BACKLOG_STATUS_ARCHIVED
	default:
		return backlogv1.BacklogStatus_BACKLOG_STATUS_UNSPECIFIED
	}
}

func statusFromProto(status backlogv1.BacklogStatus) model.Status {
	switch status {
	case backlogv1.BacklogStatus_BACKLOG_STATUS_NEW:
		return model.StatusNew
	case backlogv1.BacklogStatus_BACKLOG_STATUS_TRIAGED:
		return model.StatusTriaged
	case backlogv1.BacklogStatus_BACKLOG_STATUS_ARCHIVED:
		return model.StatusArchived
	default:
		return ""
	}
}

func updateStatusFromProto(status *backlogv1.BacklogStatus) *model.Status {
	if status == nil {
		return nil
	}
	converted := statusFromProto(*status)
	return &converted
}

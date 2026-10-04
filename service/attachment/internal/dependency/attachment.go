package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	attachmentv1 "taskmanager/common/gen/go/attachment/v1"
	"taskmanager/service/attachment/internal/model"
)

func ValidateUploadAttachment(req *attachmentv1.UploadAttachmentRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.AttachmentProfileIDRequired)
	}
	if strings.TrimSpace(req.GetOwnerType()) == "" {
		return errorcode.Error(errorcode.AttachmentOwnerTypeRequired)
	}
	if strings.TrimSpace(req.GetOwnerId()) == "" {
		return errorcode.Error(errorcode.AttachmentOwnerIDRequired)
	}
	if strings.TrimSpace(req.GetFileName()) == "" {
		return errorcode.Error(errorcode.AttachmentFileRequired)
	}
	if len(req.GetContent()) == 0 {
		return errorcode.Error(errorcode.AttachmentFileRequired)
	}
	return nil
}

func ValidateAttachmentID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.AttachmentIDRequired)
	}
	return nil
}

func UploadFromRequest(req *attachmentv1.UploadAttachmentRequest) model.Upload {
	return model.Upload{
		ProfileID:   req.GetProfileId(),
		OwnerType:   strings.TrimSpace(strings.ToLower(req.GetOwnerType())),
		OwnerID:     req.GetOwnerId(),
		FileName:    req.GetFileName(),
		ContentType: req.GetContentType(),
		Content:     req.GetContent(),
	}
}

func AttachmentFilterFromRequest(req *attachmentv1.ListAttachmentsRequest) model.Filter {
	return model.Filter{
		ProfileID: req.GetProfileId(),
		OwnerType: strings.TrimSpace(strings.ToLower(req.GetOwnerType())),
		OwnerID:   req.GetOwnerId(),
	}
}

func AttachmentToProto(item model.Attachment) *attachmentv1.Attachment {
	return &attachmentv1.Attachment{
		Id:          item.ID,
		ProfileId:   item.ProfileID,
		OwnerType:   item.OwnerType,
		OwnerId:     item.OwnerID,
		FileName:    item.FileName,
		ContentType: item.ContentType,
		Size:        item.Size,
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
}

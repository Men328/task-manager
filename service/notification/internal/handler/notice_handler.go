package handler

import (
	"context"
	"strings"

	"taskmanager/common/errorcode"
	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/notification/internal/dependency"
	"taskmanager/service/notification/internal/service"
)

type NoticeHandler struct {
	notificationv1.UnimplementedNoticeServiceServer
	notices service.NoticeService
}

func NewNoticeHandler(notices service.NoticeService) *NoticeHandler {
	return &NoticeHandler{notices: notices}
}

func (h *NoticeHandler) CreateNotice(ctx context.Context, req *notificationv1.CreateNoticeRequest) (*notificationv1.CreateNoticeResponse, error) {
	if err := dependency.ValidateCreateNotice(req); err != nil {
		return nil, err
	}

	created, err := h.notices.Create(ctx, dependency.NoticeFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &notificationv1.CreateNoticeResponse{Notice: dependency.NoticeToProto(created)}, nil
}

func (h *NoticeHandler) ListNotices(ctx context.Context, req *notificationv1.ListNoticesRequest) (*notificationv1.ListNoticesResponse, error) {
	if strings.TrimSpace(req.GetProfileId()) == "" {
		return nil, errorcode.Error(errorcode.NotificationProfileIDRequired)
	}

	notices, err := h.notices.List(ctx, dependency.FilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*notificationv1.Notice, 0, len(notices))
	for _, notice := range notices {
		out = append(out, dependency.NoticeToProto(notice))
	}
	return &notificationv1.ListNoticesResponse{Notices: out}, nil
}

func (h *NoticeHandler) CountUnreadNotices(ctx context.Context, req *notificationv1.CountUnreadNoticesRequest) (*notificationv1.CountUnreadNoticesResponse, error) {
	if strings.TrimSpace(req.GetProfileId()) == "" {
		return nil, errorcode.Error(errorcode.NotificationProfileIDRequired)
	}

	count, err := h.notices.CountUnread(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &notificationv1.CountUnreadNoticesResponse{Count: int32(count)}, nil
}

func (h *NoticeHandler) MarkNoticesRead(ctx context.Context, req *notificationv1.MarkNoticesReadRequest) (*notificationv1.MarkNoticesReadResponse, error) {
	if strings.TrimSpace(req.GetProfileId()) == "" {
		return nil, errorcode.Error(errorcode.NotificationProfileIDRequired)
	}

	marked, err := h.notices.MarkRead(ctx, req.GetProfileId(), req.GetIds())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &notificationv1.MarkNoticesReadResponse{Marked: int32(marked)}, nil
}

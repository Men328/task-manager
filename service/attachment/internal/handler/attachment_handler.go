package handler

import (
	"context"

	attachmentv1 "taskmanager/common/gen/go/attachment/v1"
	"taskmanager/service/attachment/internal/dependency"
	"taskmanager/service/attachment/internal/service"
)

type AttachmentHandler struct {
	attachmentv1.UnimplementedAttachmentServiceServer
	items service.AttachmentService
}

func NewAttachmentHandler(items service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{items: items}
}

func (h *AttachmentHandler) UploadAttachment(ctx context.Context, req *attachmentv1.UploadAttachmentRequest) (*attachmentv1.UploadAttachmentResponse, error) {
	if err := dependency.ValidateUploadAttachment(req); err != nil {
		return nil, err
	}

	created, err := h.items.Upload(ctx, dependency.UploadFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &attachmentv1.UploadAttachmentResponse{Attachment: dependency.AttachmentToProto(created)}, nil
}

func (h *AttachmentHandler) ListAttachments(ctx context.Context, req *attachmentv1.ListAttachmentsRequest) (*attachmentv1.ListAttachmentsResponse, error) {
	items, err := h.items.List(ctx, dependency.AttachmentFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*attachmentv1.Attachment, 0, len(items))
	for _, item := range items {
		out = append(out, dependency.AttachmentToProto(item))
	}
	return &attachmentv1.ListAttachmentsResponse{Attachments: out}, nil
}

func (h *AttachmentHandler) GetAttachment(ctx context.Context, req *attachmentv1.GetAttachmentRequest) (*attachmentv1.GetAttachmentResponse, error) {
	if err := dependency.ValidateAttachmentID(req.GetId()); err != nil {
		return nil, err
	}

	item, content, err := h.items.Download(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &attachmentv1.GetAttachmentResponse{
		Attachment: dependency.AttachmentToProto(item),
		Content:    content,
	}, nil
}

func (h *AttachmentHandler) DeleteAttachment(ctx context.Context, req *attachmentv1.DeleteAttachmentRequest) (*attachmentv1.DeleteAttachmentResponse, error) {
	if err := dependency.ValidateAttachmentID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.items.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &attachmentv1.DeleteAttachmentResponse{}, nil
}

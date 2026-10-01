package handler

import (
	"context"

	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/mail-provider/internal/dependency"
	"taskmanager/service/mail-provider/internal/service"
)

type MailHandler struct {
	mailv1.UnimplementedMailServiceServer
	mail service.MailService
}

func NewMailHandler(mail service.MailService) *MailHandler {
	return &MailHandler{mail: mail}
}

func (h *MailHandler) Subscribe(ctx context.Context, req *mailv1.SubscribeRequest) (*mailv1.SubscribeResponse, error) {
	if err := dependency.ValidateSubscribe(req); err != nil {
		return nil, err
	}

	subscription, err := h.mail.Subscribe(ctx, dependency.SubscribeInputFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &mailv1.SubscribeResponse{Subscription: dependency.SubscriptionToProto(subscription)}, nil
}

func (h *MailHandler) Unsubscribe(ctx context.Context, req *mailv1.UnsubscribeRequest) (*mailv1.UnsubscribeResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}
	if err := h.mail.Unsubscribe(ctx, req.GetProfileId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &mailv1.UnsubscribeResponse{}, nil
}

func (h *MailHandler) GetSubscription(ctx context.Context, req *mailv1.GetSubscriptionRequest) (*mailv1.GetSubscriptionResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}
	subscription, err := h.mail.GetSubscription(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &mailv1.GetSubscriptionResponse{Subscription: dependency.SubscriptionToProto(subscription)}, nil
}

func (h *MailHandler) HandleNotification(ctx context.Context, req *mailv1.HandleNotificationRequest) (*mailv1.HandleNotificationResponse, error) {
	if err := dependency.ValidateNotification(req.GetEmail(), req.GetHistoryId()); err != nil {
		return nil, err
	}
	accepted, err := h.mail.HandleNotification(ctx, req.GetEmail(), req.GetHistoryId(), req.GetMessageId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &mailv1.HandleNotificationResponse{Accepted: accepted}, nil
}

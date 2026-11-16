package handler

import (
	"context"

	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/dependency"
	"taskmanager/service/message/internal/service"
)

type MessageHandler struct {
	messagev1.UnimplementedMessageServiceServer
	messages service.MessageService
}

func NewMessageHandler(messages service.MessageService) *MessageHandler {
	return &MessageHandler{messages: messages}
}

func (h *MessageHandler) SendMessage(ctx context.Context, req *messagev1.SendMessageRequest) (*messagev1.SendMessageResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	message, err := h.messages.Send(ctx, req.GetProfileId(), req.GetPeerId(), req.GetBody(), req.GetClientId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &messagev1.SendMessageResponse{Message: dependency.MessageToProto(message)}, nil
}

func (h *MessageHandler) SendTaskMessage(ctx context.Context, req *messagev1.SendTaskMessageRequest) (*messagev1.SendTaskMessageResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	message, err := h.messages.SendTask(ctx, req.GetProfileId(), req.GetPeerId(), dependency.DraftFromProto(req), req.GetClientId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &messagev1.SendTaskMessageResponse{Message: dependency.MessageToProto(message)}, nil
}

func (h *MessageHandler) GetMessage(ctx context.Context, req *messagev1.GetMessageRequest) (*messagev1.GetMessageResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	message, err := h.messages.Get(ctx, req.GetProfileId(), req.GetMessageId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &messagev1.GetMessageResponse{Message: dependency.MessageToProto(message)}, nil
}

func (h *MessageHandler) ListMessages(ctx context.Context, req *messagev1.ListMessagesRequest) (*messagev1.ListMessagesResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	messages, hasMore, err := h.messages.ListMessages(ctx, req.GetProfileId(), req.GetPeerId(), dependency.BeforeFromProto(req.GetBefore()), int(req.GetLimit()))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &messagev1.ListMessagesResponse{Messages: dependency.MessagesToProto(messages), HasMore: hasMore}, nil
}

func (h *MessageHandler) ListConversations(ctx context.Context, req *messagev1.ListConversationsRequest) (*messagev1.ListConversationsResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	conversations, err := h.messages.ListConversations(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &messagev1.ListConversationsResponse{Conversations: dependency.ConversationsToProto(conversations)}, nil
}

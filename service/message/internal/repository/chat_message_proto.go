package repository

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/model"
)

func chatMessageProto(m model.Message) *messagev1.ChatMessage {
	out := &messagev1.ChatMessage{
		Id:             m.ID,
		ConversationId: m.ConversationID,
		SenderId:       m.SenderID,
		RecipientId:    m.RecipientID,
		Body:           m.Body,
		ClientId:       m.ClientID,
		CreatedAt:      timestamppb.New(m.CreatedAt),
	}
	if m.Task != nil {
		out.Task = &messagev1.TaskRef{
			TaskId:         m.Task.TaskID,
			Title:          m.Task.Title,
			StatusName:     m.Task.StatusName,
			StatusCategory: m.Task.StatusCategory,
			StatusColor:    m.Task.StatusColor,
		}
		if !m.Task.StatusChangedAt.IsZero() {
			out.Task.StatusChangedAt = timestamppb.New(m.Task.StatusChangedAt)
		}
	}
	return out
}

func EncodeMessageEvent(message model.Message) ([]byte, error) {
	value, err := protojson.Marshal(chatMessageProto(message))
	if err != nil {
		return nil, fmt.Errorf("mã hoá tin nhắn: %w", err)
	}
	return value, nil
}

package dependency

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/model"
)

func ValidateProfileID(profileID string) error {
	if strings.TrimSpace(profileID) == "" {
		return errorcode.Error(errorcode.CommonInvalidArgument)
	}
	return nil
}

func MessageToProto(m model.Message) *messagev1.ChatMessage {
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
		out.Task = TaskRefToProto(*m.Task)
	}
	return out
}

func TaskRefToProto(ref model.TaskRef) *messagev1.TaskRef {
	out := &messagev1.TaskRef{
		TaskId:         ref.TaskID,
		Title:          ref.Title,
		StatusName:     ref.StatusName,
		StatusCategory: ref.StatusCategory,
		StatusColor:    ref.StatusColor,
	}
	if !ref.StatusChangedAt.IsZero() {
		out.StatusChangedAt = timestamppb.New(ref.StatusChangedAt)
	}
	return out
}

func DraftFromProto(req *messagev1.SendTaskMessageRequest) model.TaskDraft {
	draft := model.TaskDraft{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Priority:    req.GetPriority(),
	}
	if req.GetDueAt() != nil {
		due := req.GetDueAt().AsTime().UTC()
		draft.DueAt = &due
	}
	return draft
}

func MessagesToProto(messages []model.Message) []*messagev1.ChatMessage {
	out := make([]*messagev1.ChatMessage, 0, len(messages))
	for _, m := range messages {
		out = append(out, MessageToProto(m))
	}
	return out
}

func ConversationsToProto(conversations []model.Conversation) []*messagev1.Conversation {
	out := make([]*messagev1.Conversation, 0, len(conversations))
	for _, c := range conversations {
		out = append(out, &messagev1.Conversation{
			Id:          c.ID,
			PeerId:      c.PeerID,
			LastMessage: MessageToProto(c.LastMessage),
		})
	}
	return out
}

func BeforeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

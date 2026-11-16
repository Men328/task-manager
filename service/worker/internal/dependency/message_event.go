package dependency

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	messagev1 "taskmanager/common/gen/go/message/v1"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/worker/internal/model"
)

var eventDecoder = protojson.UnmarshalOptions{DiscardUnknown: true}

func DecodeMessageEvent(value []byte) (model.Message, error) {
	var event messagev1.ChatMessage
	if err := eventDecoder.Unmarshal(value, &event); err != nil {
		return model.Message{}, fmt.Errorf("%w: giải mã ChatMessage: %v", model.ErrPermanent, err)
	}

	message := model.Message{
		ID:             event.GetId(),
		ConversationID: event.GetConversationId(),
		SenderID:       event.GetSenderId(),
		RecipientID:    event.GetRecipientId(),
		Body:           event.GetBody(),
		ClientID:       event.GetClientId(),
	}
	if event.GetCreatedAt() != nil {
		message.CreatedAt = event.GetCreatedAt().AsTime()
	}
	if ref := event.GetTask(); ref != nil && ref.GetTaskId() != "" {
		message.Task = &model.TaskRef{
			TaskID:         ref.GetTaskId(),
			Title:          ref.GetTitle(),
			StatusName:     ref.GetStatusName(),
			StatusCategory: ref.GetStatusCategory(),
			StatusColor:    ref.GetStatusColor(),
		}
		if ref.GetStatusChangedAt() != nil {
			message.Task.StatusChangedAt = ref.GetStatusChangedAt().AsTime()
		}
	}
	return message, nil
}

func DecodeTaskStatusChanged(value []byte) (model.TaskStatusChanged, error) {
	var event taskv1.TaskStatusChangedEvent
	if err := eventDecoder.Unmarshal(value, &event); err != nil {
		return model.TaskStatusChanged{}, fmt.Errorf("%w: giải mã TaskStatusChangedEvent: %v", model.ErrPermanent, err)
	}
	out := model.TaskStatusChanged{
		EventID:        event.GetEventId(),
		TaskID:         event.GetTaskId(),
		OwnerID:        event.GetProfileId(),
		Title:          event.GetTitle(),
		RequestedBy:    event.GetRequestedBy(),
		MessageID:      event.GetSourceMessageId(),
		FromStatusName: event.GetFromStatus().GetName(),
		ToStatusName:   event.GetToStatus().GetName(),
		ToCategory:     categoryName(event.GetToStatus().GetCategory()),
		ToColor:        event.GetToStatus().GetColor(),
	}
	if event.GetChangedAt() != nil {
		out.ChangedAt = event.GetChangedAt().AsTime()
	}
	return out, nil
}

func categoryName(category taskv1.TaskStatusCategory) string {
	name := strings.TrimPrefix(category.String(), "TASK_STATUS_CATEGORY_")
	if name == "UNSPECIFIED" || name == category.String() {
		return ""
	}
	return strings.ToLower(name)
}

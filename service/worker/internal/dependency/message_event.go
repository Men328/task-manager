package dependency

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	messagev1 "taskmanager/common/gen/go/message/v1"
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
	return message, nil
}

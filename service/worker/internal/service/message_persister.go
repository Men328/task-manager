package service

import (
	"context"
	"fmt"
	"strings"

	"taskmanager/service/worker/internal/model"
)

type messagePersister struct {
	store MessageStore
}

func NewMessagePersister(store MessageStore) MessagePersister {
	return &messagePersister{store: store}
}

func (p *messagePersister) Persist(ctx context.Context, message model.Message) error {
	if err := validate(message); err != nil {
		return err
	}
	return p.store.Save(ctx, message)
}

func validate(m model.Message) error {
	required := map[string]string{
		"id":              m.ID,
		"conversation_id": m.ConversationID,
		"sender_id":       m.SenderID,
		"recipient_id":    m.RecipientID,
		"body":            m.Body,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: thiếu %s", model.ErrPermanent, field)
		}
	}
	if m.SenderID == m.RecipientID {
		return fmt.Errorf("%w: sender trùng recipient", model.ErrPermanent)
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("%w: thiếu created_at", model.ErrPermanent)
	}
	return nil
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"taskmanager/service/worker/internal/model"
)

type taskEventHandler struct {
	store       MessageStore
	broadcaster MessageBroadcaster
	notifier    TaskNotifier
}

func NewTaskEventHandler(store MessageStore, broadcaster MessageBroadcaster, notifier TaskNotifier) TaskEventHandler {
	return &taskEventHandler{store: store, broadcaster: broadcaster, notifier: notifier}
}

func (h *taskEventHandler) Handle(ctx context.Context, event model.TaskStatusChanged) error {
	if strings.TrimSpace(event.TaskID) == "" || strings.TrimSpace(event.RequestedBy) == "" || strings.TrimSpace(event.MessageID) == "" {
		return fmt.Errorf("%w: sự kiện task thiếu task_id/requested_by/source_message_id", model.ErrPermanent)
	}
	if event.ChangedAt.IsZero() {
		return fmt.Errorf("%w: sự kiện task thiếu changed_at", model.ErrPermanent)
	}

	message, updated, err := h.store.UpdateTaskStatus(ctx, event)
	if err != nil {
		return err
	}
	if updated {
		if err := h.broadcaster.BroadcastUpdated(ctx, message); err != nil {
			slog.Warn("broadcast cập nhật thẻ task thất bại", "message_id", message.ID, "error", err)
		}
	} else {
		slog.Info("không cập nhật thẻ task: tin nhắn chưa lưu hoặc sự kiện cũ hơn", "message_id", event.MessageID, "task_id", event.TaskID)
	}

	if event.RequestedBy == event.OwnerID {
		return nil
	}
	return h.notifier.TaskStatusChanged(ctx, event)
}

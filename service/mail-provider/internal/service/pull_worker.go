package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type pullWorker struct {
	mail        MailService
	source      NotificationSource
	opts        Options
	onHeartbeat func(emptyPulls int, idle time.Duration)
	now         func() time.Time
}

func NewPullWorker(mail MailService, source NotificationSource, opts Options) Worker {
	options := opts.withDefaults()
	return &pullWorker{
		mail:   mail,
		source: source,
		opts:   options,
		now:    time.Now,
		onHeartbeat: func(emptyPulls int, idle time.Duration) {
			slog.Info("pull đang chạy nhưng chưa có notice nào",
				"subscription", options.PullSubscription,
				"empty_pulls", emptyPulls,
				"idle_for", idle.Round(time.Second),
			)
		},
	}
}

func (w *pullWorker) Run(ctx context.Context) error {
	emptyPulls := 0
	lastActivity := w.now()

	for {
		if ctx.Err() != nil {
			return nil
		}

		notices, err := w.source.Receive(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("pull notice từ pubsub thất bại", "error", err)
			if !sleepContext(ctx, w.opts.PullRetryDelay) {
				return nil
			}
			continue
		}

		if len(notices) == 0 {
			emptyPulls++
			if w.opts.HeartbeatInterval > 0 {
				now := w.now()
				if idle := now.Sub(lastActivity); idle >= w.opts.HeartbeatInterval {
					if w.onHeartbeat != nil {
						w.onHeartbeat(emptyPulls, idle)
					}
					emptyPulls = 0
					lastActivity = now
				}
			}
			continue
		}

		emptyPulls = 0
		lastActivity = w.now()
		w.process(ctx, notices)
	}
}

func (w *pullWorker) process(ctx context.Context, notices []model.PulledNotice) {
	if len(notices) == 0 {
		return
	}

	ackIDs := make([]string, 0, len(notices))
	for _, item := range notices {
		if item.AckID == "" {
			continue
		}

		if item.Notice.EmailAddress == "" || item.Notice.HistoryID == "" {
			slog.Warn("bỏ qua notice pull không hợp lệ", "ack_id", item.AckID, "payload", item.Raw)
			ackIDs = append(ackIDs, item.AckID)
			continue
		}

		accepted, err := w.mail.HandleNotification(ctx, item.Notice.EmailAddress, item.Notice.HistoryID, item.Notice.MessageID)
		if err != nil && !errors.Is(err, model.ErrInvalidPayload) {
			slog.Error("xử lý notice pull thất bại, sẽ để pubsub gửi lại",
				"email", item.Notice.EmailAddress,
				"history_id", item.Notice.HistoryID,
				"error", err,
			)
			continue
		}

		slog.Info("nhận notice gmail qua pull",
			"email", item.Notice.EmailAddress,
			"history_id", item.Notice.HistoryID,
			"message_id", item.Notice.MessageID,
			"accepted", accepted,
		)
		ackIDs = append(ackIDs, item.AckID)
	}

	if len(ackIDs) == 0 {
		return
	}
	if err := w.source.Acknowledge(ctx, ackIDs); err != nil {
		slog.Error("ack notice thất bại", "count", len(ackIDs), "error", err)
	}
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

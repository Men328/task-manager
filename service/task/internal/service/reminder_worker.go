package service

import (
	"context"
	"log/slog"
	"time"

	"taskmanager/service/task/internal/model"
)

type ReminderConfig struct {
	Interval time.Duration
	Batch    int
	MaxLate  time.Duration
}

type ReminderWorker struct {
	store  ReminderStore
	sender NoticeSender
	cfg    ReminderConfig
	now    func() time.Time
}

func NewReminderWorker(store ReminderStore, sender NoticeSender, cfg ReminderConfig) *ReminderWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.Batch <= 0 {
		cfg.Batch = 100
	}
	return &ReminderWorker{store: store, sender: sender, cfg: cfg, now: time.Now}
}

type ReminderRunStats struct {
	Sent    int
	Skipped int
	Failed  int
}

func (w *ReminderWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		w.drain(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *ReminderWorker) drain(ctx context.Context) {
	for ctx.Err() == nil {
		stats, claimed, err := w.RunOnce(ctx)
		if err != nil {
			slog.Error("quét nhắc việc thất bại", "error", err)
			return
		}
		if stats.Sent+stats.Skipped+stats.Failed > 0 {
			slog.Info("đã xử lý nhắc việc", "sent", stats.Sent, "skipped", stats.Skipped, "failed", stats.Failed)
		}
		if claimed < w.cfg.Batch || stats.Failed > 0 {
			return
		}
	}
}

func (w *ReminderWorker) RunOnce(ctx context.Context) (ReminderRunStats, int, error) {
	var stats ReminderRunStats
	tasks, err := w.store.ClaimDueReminders(ctx, w.cfg.Batch)
	if err != nil {
		return stats, 0, err
	}
	cutoff := w.now().Add(-w.cfg.MaxLate)
	for _, t := range tasks {
		if w.cfg.MaxLate > 0 && t.RemindAt != nil && t.RemindAt.Before(cutoff) {
			stats.Skipped++
			continue
		}
		if err := w.sender.SendTaskReminder(ctx, t); err != nil {
			stats.Failed++
			slog.Warn("gửi nhắc việc thất bại, sẽ thử lại", "task_id", t.ID, "error", err)
			w.release(ctx, t)
			continue
		}
		stats.Sent++
	}
	return stats, len(tasks), nil
}

func (w *ReminderWorker) release(ctx context.Context, t model.Task) {
	if t.RemindedAt == nil {
		return
	}
	if err := w.store.ReleaseReminder(context.WithoutCancel(ctx), t.ID, *t.RemindedAt); err != nil {
		slog.Error("trả lại nhắc việc thất bại", "task_id", t.ID, "error", err)
	}
}

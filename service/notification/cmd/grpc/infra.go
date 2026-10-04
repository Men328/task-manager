package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"taskmanager/service/notification/internal/config"
	"taskmanager/service/notification/internal/repository"
	"taskmanager/service/notification/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: notification dùng repository in-memory, dữ liệu sẽ mất khi restart")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("khởi tạo postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})
	slog.Info("notification dùng repository postgres")
	return pool, nil
}

func newNoticeRepository(pool *pgxpool.Pool) service.NoticeRepository {
	if pool == nil {
		return repository.NewInMemoryNoticeRepository()
	}
	return repository.NewPostgresNoticeRepository(pool)
}

func newNoticePublisher(cfg config.Config) service.NoticePublisher {
	if !cfg.SoketiConfigured() {
		slog.Warn("thiếu SOKETI_APP_KEY/SOKETI_APP_SECRET: notice không được đẩy qua websocket")
		return repository.NewNoopPublisher()
	}

	slog.Info("notification đẩy notice qua soketi",
		"base_url", cfg.SoketiBaseURL,
		"app_id", cfg.SoketiAppID,
		"channel_prefix", cfg.ChannelPrefix(),
	)
	return repository.NewSoketiPublisher(
		cfg.SoketiBaseURL,
		cfg.SoketiAppID,
		cfg.SoketiAppKey,
		cfg.SoketiAppSecret,
		cfg.ChannelPrefix(),
		cfg.SoketiTimeout,
	)
}

func warnSoketiConfig(cfg config.Config) {
	if !cfg.SoketiConfigured() {
		slog.Warn("SOKETI_APP_KEY/SOKETI_APP_SECRET trống: notice không được đẩy qua websocket")
	}
}

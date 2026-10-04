package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"taskmanager/service/attachment/internal/config"
	"taskmanager/service/attachment/internal/repository"
	"taskmanager/service/attachment/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: attachment dùng repository in-memory, dữ liệu sẽ mất khi restart")
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
	slog.Info("attachment dùng repository postgres")
	return pool, nil
}

func newAttachmentRepository(pool *pgxpool.Pool) service.AttachmentRepository {
	if pool == nil {
		return repository.NewInMemoryAttachmentRepository()
	}
	return repository.NewPostgresAttachmentRepository(pool)
}

func newBlobStore(cfg config.Config) (service.BlobStore, error) {
	if !cfg.StorageConfigured() {
		slog.Warn("thiếu S3_ENDPOINT: attachment dùng blob store in-memory (chỉ hợp lệ khi dev)")
		return repository.NewMemoryBlobStore(), nil
	}

	store, err := repository.NewS3BlobStore(
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3Bucket,
		cfg.S3Region,
		cfg.S3UseSSL,
		cfg.S3Timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("khởi tạo S3 blob store: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.S3Timeout)
		lastErr = store.EnsureBucket(ctx)
		cancel()
		if lastErr == nil {
			break
		}
		slog.Warn("chờ object storage sẵn sàng", "endpoint", cfg.S3Endpoint, "attempt", attempt, "error", lastErr)
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		slog.Error("tạo bucket S3 thất bại", "bucket", cfg.S3Bucket, "error", lastErr)
	}

	slog.Info("attachment lưu tệp trên object storage S3",
		"endpoint", cfg.S3Endpoint,
		"bucket", cfg.S3Bucket,
	)
	return store, nil
}

func newAttachmentService(items service.AttachmentRepository, blobs service.BlobStore, cfg config.Config) service.AttachmentService {
	return service.NewAttachmentService(items, blobs, cfg.MaxAttachmentBytes)
}

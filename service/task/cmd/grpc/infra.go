package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"taskmanager/service/task/internal/config"
	"taskmanager/service/task/internal/repository"
	"taskmanager/service/task/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: task dùng repository in-memory, dữ liệu sẽ mất khi restart")
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
	slog.Info("task dùng repository postgres")
	return pool, nil
}

func newTaskRepository(pool *pgxpool.Pool) service.TaskRepository {
	if pool == nil {
		return repository.NewInMemoryTaskRepository()
	}
	return repository.NewPostgresTaskRepository(pool)
}

func newStatusRepository(pool *pgxpool.Pool) service.StatusRepository {
	if pool == nil {
		return repository.NewInMemoryStatusRepository()
	}
	return repository.NewPostgresStatusRepository(pool)
}

func newTransitionRepository(pool *pgxpool.Pool) service.TransitionRepository {
	if pool == nil {
		return repository.NewInMemoryTransitionRepository()
	}
	return repository.NewPostgresTransitionRepository(pool)
}

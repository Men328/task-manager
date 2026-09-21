package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"taskmanager/service/identity/internal/config"
	"taskmanager/service/identity/internal/repository"
	"taskmanager/service/identity/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: identity dùng repository in-memory, dữ liệu sẽ mất khi restart")
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
	slog.Info("identity dùng repository postgres")
	return pool, nil
}

func newProfileRepository(pool *pgxpool.Pool) service.ProfileRepository {
	if pool == nil {
		return repository.NewInMemoryProfileRepository()
	}
	return repository.NewPostgresProfileRepository(pool)
}

func newAuthProviderRepository(pool *pgxpool.Pool) service.AuthProviderRepository {
	if pool == nil {
		return repository.NewInMemoryAuthProviderRepository()
	}
	return repository.NewPostgresAuthProviderRepository(pool)
}

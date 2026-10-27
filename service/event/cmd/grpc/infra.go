package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/event/internal/config"
	"taskmanager/service/event/internal/repository"
	"taskmanager/service/event/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: event dùng repository in-memory, dữ liệu sẽ mất khi restart")
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
	slog.Info("event dùng repository postgres")
	return pool, nil
}

func newEventRepository(pool *pgxpool.Pool) service.EventRepository {
	if pool == nil {
		return repository.NewInMemoryEventRepository()
	}
	return repository.NewPostgresEventRepository(pool)
}

func newProjectConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(cfg.ProjectDialTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC project (%s): %w", cfg.ProjectDialTarget(), err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newProjectServiceClient(conn *grpc.ClientConn) projectv1.ProjectServiceClient {
	return projectv1.NewProjectServiceClient(conn)
}

func newProjectChecker(client projectv1.ProjectServiceClient, cfg config.Config) service.ProjectChecker {
	return repository.NewProjectChecker(client, cfg.ProjectTimeout)
}

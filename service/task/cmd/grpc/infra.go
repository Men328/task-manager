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

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	projectv1 "taskmanager/common/gen/go/project/v1"
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

func newNotificationConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(cfg.NotificationDialTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC notification (%s): %w", cfg.NotificationDialTarget(), err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newNoticeServiceClient(conn *grpc.ClientConn) notificationv1.NoticeServiceClient {
	return notificationv1.NewNoticeServiceClient(conn)
}

func newReminderWorker(pool *pgxpool.Pool, client notificationv1.NoticeServiceClient, cfg config.Config) *service.ReminderWorker {
	if !cfg.ReminderEnabled {
		slog.Info("nhắc việc đang tắt (REMINDER_ENABLED=false)")
		return nil
	}
	if pool == nil {
		slog.Warn("không chạy nhắc việc: cần DATABASE_URL (repository in-memory không hỗ trợ)")
		return nil
	}
	sender := repository.NewNotificationClient(client, cfg.NotificationTimeout, cfg.ReminderLocation())
	return service.NewReminderWorker(repository.NewPostgresTaskRepository(pool), sender, service.ReminderConfig{
		Interval: cfg.ReminderInterval,
		Batch:    cfg.ReminderBatch,
		MaxLate:  cfg.ReminderMaxLate,
	})
}

func startReminderWorker(lc fx.Lifecycle, worker *service.ReminderWorker) {
	if worker == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				defer close(done)
				if err := worker.Run(ctx); err != nil {
					slog.Error("worker nhắc việc dừng bất thường", "error", err)
				}
			}()
			slog.Info("đã bật worker nhắc việc")
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			select {
			case <-done:
			case <-stopCtx.Done():
			}
			return nil
		},
	})
}

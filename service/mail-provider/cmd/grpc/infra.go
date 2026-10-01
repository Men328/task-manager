package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	taskv1 "taskmanager/common/gen/go/task/v1"
	workspacev1 "taskmanager/common/gen/go/workspace/v1"
	"taskmanager/service/mail-provider/internal/config"
	"taskmanager/service/mail-provider/internal/model"
	"taskmanager/service/mail-provider/internal/repository"
	"taskmanager/service/mail-provider/internal/service"
)

func newSubscriptionRepository(pool *pgxpool.Pool) service.SubscriptionRepository {
	if pool == nil {
		return repository.NewInMemorySubscriptionRepository()
	}
	return repository.NewPostgresSubscriptionRepository(pool)
}

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: mail-provider dùng repository in-memory, subscription sẽ mất khi restart")
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
	slog.Info("mail-provider dùng repository postgres cho subscription")
	return pool, nil
}

func newQueue(cfg config.Config) *service.Queue {
	return service.NewQueue(cfg.QueueSize)
}

func newGmailClient(cfg config.Config) service.GmailClient {
	return repository.NewGmailClient(cfg.GmailBaseURL, cfg.GmailTimeout, cfg.GmailMaxBodyBytes)
}

func newTokenRefresher(cfg config.Config) service.TokenRefresher {
	return repository.NewTokenRefresher(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleTokenURL, 15*time.Second)
}

func newMailAnalyzer(cfg config.Config) service.MailAnalyzer {
	return repository.NewDeepSeekAnalyzer(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel, cfg.DeepSeekTimeout)
}

func newTaskCreator(client taskv1.TaskServiceClient, cfg config.Config) service.TaskCreator {
	return repository.NewTaskClient(client, cfg.TaskTimeout)
}

func newWorkspaceResolver(client workspacev1.WorkspaceServiceClient, cfg config.Config) service.WorkspaceResolver {
	return repository.NewWorkspaceResolver(client, cfg.TaskTimeout)
}

func newTaskConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.TaskDialTarget(), "task")
}

func newWorkspaceConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.WorkspaceDialTarget(), "workspace")
}

func dialGRPC(lc fx.Lifecycle, target string, name string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC %s (%s): %w", name, target, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newTaskServiceClient(conn *grpc.ClientConn) taskv1.TaskServiceClient {
	return taskv1.NewTaskServiceClient(conn)
}

func newWorkspaceServiceClient(conn *grpc.ClientConn) workspacev1.WorkspaceServiceClient {
	return workspacev1.NewWorkspaceServiceClient(conn)
}

func newNotificationSource(cfg config.Config) service.NotificationSource {
	return repository.NewPubSubPullClient(cfg.PubSubBaseURL, cfg.PullSubscription, cfg.PullMaxMessages, cfg.PullTimeout)
}

func newOptions(cfg config.Config) service.Options {
	return service.Options{
		TopicName:         cfg.PubSubTopic,
		LabelIDs:          cfg.WatchLabelIDs,
		RenewInterval:     cfg.WatchRenewInterval,
		RenewThreshold:    cfg.WatchRenewThreshold,
		WorkerCount:       cfg.WorkerCount,
		DefaultPriority:   parsePriority(cfg.DefaultPriority),
		AnalyzerEnabled:   cfg.AnalyzerConfigured(),
		PullRetryDelay:    cfg.PullRetryDelay,
		PullSubscription:  cfg.PullSubscription,
		HeartbeatInterval: cfg.PullHeartbeat,
	}
}

func parsePriority(value string) model.Priority {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return model.PriorityLow
	case "medium", "normal":
		return model.PriorityMedium
	case "high":
		return model.PriorityHigh
	case "urgent", "critical":
		return model.PriorityUrgent
	default:
		return model.PriorityMedium
	}
}

func warnMailConfig(cfg config.Config) {
	if !cfg.PubSubConfigured() {
		slog.Warn("thiếu MAIL_PUBSUB_TOPIC: Subscribe sẽ trả MAIL_NOT_CONFIGURED")
	}
	if !cfg.AnalyzerConfigured() {
		slog.Warn("thiếu DEEPSEEK_API_KEY: notice vẫn được ghi nhận nhưng không tạo task")
	}
	if !cfg.TokenRefreshConfigured() {
		slog.Warn("thiếu GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET: không refresh được access token")
	}
	if cfg.PullConfigured() {
		slog.Info("bật chế độ pull notice từ pubsub", "subscription", cfg.PullSubscription)
	}
	if !cfg.PubSubConfigured() || !cfg.AnalyzerConfigured() {
		return
	}
	slog.Info("mail provider đã sẵn sàng",
		"topic", cfg.PubSubTopic,
		"workers", cfg.WorkerCount,
		"deepseek_model", cfg.DeepSeekModel,
	)
}

func startWorker(lc fx.Lifecycle, worker service.Worker) {
	runWorker(lc, "queue notice", worker)
}

func startPullWorker(lc fx.Lifecycle, cfg config.Config, worker service.Worker) {
	if !cfg.PullConfigured() {
		slog.Info("không bật pull notice: thiếu MAIL_PULL_SUBSCRIPTION")
		return
	}
	runWorker(lc, "pull notice", worker)
}

func runWorker(lc fx.Lifecycle, name string, worker service.Worker) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := worker.Run(ctx); err != nil {
					slog.Error("worker dừng bất thường", "worker", name, "error", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

func setupLogger(cfg config.Config) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()})))
}

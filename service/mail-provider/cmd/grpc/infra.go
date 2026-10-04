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

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	eventv1 "taskmanager/common/gen/go/event/v1"
	notificationv1 "taskmanager/common/gen/go/notification/v1"
	taskv1 "taskmanager/common/gen/go/task/v1"
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

func newBlobStore(cfg config.Config) service.BlobStore {
	if !cfg.StorageConfigured() {
		slog.Warn("thiếu S3_ENDPOINT: mail-provider không lưu email gốc lên object storage")
		return repository.NewNoopBlobStore()
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
		slog.Error("khởi tạo S3 client thất bại, tắt archive", "error", err)
		return repository.NewNoopBlobStore()
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
		slog.Error("tạo bucket S3 thất bại, tắt archive", "bucket", cfg.S3Bucket, "error", lastErr)
		return repository.NewNoopBlobStore()
	}

	slog.Info("mail-provider lưu email gốc trên object storage S3",
		"endpoint", cfg.S3Endpoint,
		"bucket", cfg.S3Bucket,
	)
	return store
}

func newTaskCreator(client taskv1.TaskServiceClient, cfg config.Config) service.TaskCreator {
	return repository.NewTaskClient(client, cfg.TaskTimeout)
}

func newScheduleCreator(client calendarv1.CalendarServiceClient, cfg config.Config) service.ScheduleCreator {
	return repository.NewScheduleClient(client, cfg.ScheduleTimeout)
}

func newEventCreator(client eventv1.EventServiceClient, cfg config.Config) service.EventCreator {
	return repository.NewEventClient(client, cfg.EventTimeout)
}

func newBacklogCreator(client backlogv1.BacklogServiceClient, cfg config.Config) service.BacklogCreator {
	return repository.NewBacklogClient(client, cfg.BacklogTimeout)
}

func newNoticePublisher(client notificationv1.NoticeServiceClient, cfg config.Config) service.NoticePublisher {
	return repository.NewNotificationClient(client, cfg.NotificationTimeout)
}

func newTaskConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.TaskDialTarget(), "task")
}

func newCalendarConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.ScheduleDialTarget(), "calendar")
}

func newEventConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.EventDialTarget(), "event")
}

func newBacklogConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.BacklogDialTarget(), "backlog")
}

func newNotificationConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dialGRPC(lc, cfg.NotificationDialTarget(), "notification")
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

func newCalendarServiceClient(conn *grpc.ClientConn) calendarv1.CalendarServiceClient {
	return calendarv1.NewCalendarServiceClient(conn)
}

func newEventServiceClient(conn *grpc.ClientConn) eventv1.EventServiceClient {
	return eventv1.NewEventServiceClient(conn)
}

func newBacklogServiceClient(conn *grpc.ClientConn) backlogv1.BacklogServiceClient {
	return backlogv1.NewBacklogServiceClient(conn)
}

func newNotificationServiceClient(conn *grpc.ClientConn) notificationv1.NoticeServiceClient {
	return notificationv1.NewNoticeServiceClient(conn)
}

func newNotificationSource(cfg config.Config) service.NotificationSource {
	return repository.NewPubSubPullClient(cfg.PubSubBaseURL, cfg.PullSubscription, cfg.PullMaxMessages, cfg.PullTimeout)
}

func newOptions(cfg config.Config) service.Options {
	return service.Options{
		TopicName:           cfg.PubSubTopic,
		LabelIDs:            cfg.WatchLabelIDs,
		RenewInterval:       cfg.WatchRenewInterval,
		RenewThreshold:      cfg.WatchRenewThreshold,
		WorkerCount:         cfg.WorkerCount,
		DefaultPriority:     parsePriority(cfg.DefaultPriority),
		AnalyzerEnabled:     cfg.AnalyzerConfigured(),
		RuleFallbackEnabled: cfg.RuleFallbackEnabled,
		ArchiveEnabled:      cfg.ArchiveConfigured(),
		MaxAttachments:      cfg.MaxAttachments,
		MaxAttachmentBytes:  cfg.MaxAttachmentBytes,
		PullRetryDelay:      cfg.PullRetryDelay,
		PullSubscription:    cfg.PullSubscription,
		HeartbeatInterval:   cfg.PullHeartbeat,
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
		slog.Warn("thiếu DEEPSEEK_API_KEY: mail-provider chỉ dùng rule keyword để phân loại")
	}
	if !cfg.TokenRefreshConfigured() {
		slog.Warn("thiếu GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET: không refresh được access token")
	}
	if !cfg.StorageConfigured() {
		slog.Warn("thiếu S3_ENDPOINT: email gốc không được lưu trữ")
	}
	if cfg.PullConfigured() {
		slog.Info("bật chế độ pull notice từ pubsub", "subscription", cfg.PullSubscription)
	}
	if !cfg.PubSubConfigured() {
		return
	}
	slog.Info("mail provider đã sẵn sàng",
		"topic", cfg.PubSubTopic,
		"workers", cfg.WorkerCount,
		"deepseek_model", cfg.DeepSeekModel,
		"rule_fallback", cfg.RuleFallbackEnabled,
		"archive", cfg.ArchiveConfigured(),
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

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	notificationv1 "taskmanager/common/gen/go/notification/v1"

	"taskmanager/service/worker/internal/config"
	"taskmanager/service/worker/internal/handler"
	"taskmanager/service/worker/internal/model"
	"taskmanager/service/worker/internal/repository"
	"taskmanager/service/worker/internal/service"
)

func setupLogger(cfg config.Config) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()})))
}

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL là bắt buộc: worker không có nơi lưu tin nhắn")
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
	return pool, nil
}

func newMessageStore(pool *pgxpool.Pool) service.MessageStore {
	return repository.NewPostgresMessageStore(pool)
}

func newKafkaReader(lc fx.Lifecycle, cfg config.Config, topic, groupID string) (*kafka.Reader, error) {
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS là bắt buộc")
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0,
		StartOffset:    kafka.FirstOffset,
	})
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return reader.Close()
		},
	})
	return reader, nil
}

func retryPolicy(cfg config.Config) handler.RetryPolicy {
	return handler.RetryPolicy{
		Initial:        cfg.RetryInitialBackoff,
		Max:            cfg.RetryMaxBackoff,
		PersistTimeout: cfg.PersistTimeout,
	}
}

func newChatConsumer(lc fx.Lifecycle, cfg config.Config, persister service.MessagePersister) (*handler.Consumer, error) {
	reader, err := newKafkaReader(lc, cfg, cfg.KafkaMessageTopic, cfg.KafkaMessageGroupID)
	if err != nil {
		return nil, err
	}
	name := cfg.KafkaMessageTopic + "/" + cfg.KafkaMessageGroupID
	return handler.NewConsumer(name, reader, handler.NewChatMessageProcessor(persister), retryPolicy(cfg)), nil
}

func newTaskEventConsumer(lc fx.Lifecycle, cfg config.Config, events service.TaskEventHandler) (*handler.Consumer, error) {
	reader, err := newKafkaReader(lc, cfg, cfg.KafkaTaskEventsTopic, cfg.KafkaTaskEventsGroupID)
	if err != nil {
		return nil, err
	}
	name := cfg.KafkaTaskEventsTopic + "/" + cfg.KafkaTaskEventsGroupID
	return handler.NewConsumer(name, reader, handler.NewTaskEventProcessor(events), retryPolicy(cfg)), nil
}

func newNotificationConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(cfg.NotificationGRPCDialAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC notification (%s): %w", cfg.NotificationGRPCDialAddr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newTaskNotifier(conn *grpc.ClientConn, cfg config.Config) service.TaskNotifier {
	return repository.NewNotificationClient(notificationv1.NewNoticeServiceClient(conn), cfg.UpstreamTimeout)
}

type noopBroadcaster struct{}

func (noopBroadcaster) BroadcastUpdated(context.Context, model.Message) error { return nil }

func newBroadcaster(cfg config.Config) service.MessageBroadcaster {
	if !cfg.SoketiConfigured() {
		slog.Warn("thiếu SOKETI_*: cập nhật thẻ task không được đẩy realtime")
		return noopBroadcaster{}
	}
	return repository.NewSoketiBroadcaster(cfg.SoketiBaseURL, cfg.SoketiAppID, cfg.SoketiAppKey, cfg.SoketiAppSecret,
		cfg.SoketiChatChannelPrefix, cfg.SoketiTimeout)
}

func runConsumers(lc fx.Lifecycle, consumers []*handler.Consumer) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			for _, consumer := range consumers {
				slog.Info("worker consume kafka", "consumer", consumer.Name())
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := consumer.Run(ctx); err != nil {
						slog.Error("consumer dừng bất thường", "consumer", consumer.Name(), "error", err)
					}
				}()
			}
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-stopCtx.Done():
			}
			return nil
		},
	})
}

func newHealthServer(pool *pgxpool.Pool) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"degraded","service":"worker"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","service":"worker"}`))
	})
	return &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func serveHealth(lc fx.Lifecycle, cfg config.Config, server *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", cfg.HealthAddr)
			if err != nil {
				return fmt.Errorf("listen health %s: %w", cfg.HealthAddr, err)
			}
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("serve health", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

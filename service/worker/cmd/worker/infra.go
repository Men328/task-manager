package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"

	"taskmanager/service/worker/internal/config"
	"taskmanager/service/worker/internal/handler"
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

func newMessageReader(lc fx.Lifecycle, cfg config.Config) (*kafka.Reader, error) {
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS là bắt buộc")
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		GroupID:        cfg.KafkaMessageGroupID,
		Topic:          cfg.KafkaMessageTopic,
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

func newMessageConsumer(reader *kafka.Reader, persister service.MessagePersister, cfg config.Config) *handler.MessageConsumer {
	return handler.NewMessageConsumer(reader, persister, handler.RetryPolicy{
		Initial:        cfg.RetryInitialBackoff,
		Max:            cfg.RetryMaxBackoff,
		PersistTimeout: cfg.PersistTimeout,
	})
}

func runMessageConsumer(lc fx.Lifecycle, cfg config.Config, consumer *handler.MessageConsumer) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			slog.Info("worker consume kafka",
				"service", cfg.ServiceName,
				"brokers", cfg.KafkaBrokers,
				"topic", cfg.KafkaMessageTopic,
				"group_id", cfg.KafkaMessageGroupID,
			)
			go func() {
				defer close(done)
				if err := consumer.Run(ctx); err != nil {
					slog.Error("consumer dừng bất thường", "error", err)
				}
			}()
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

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

	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/message/internal/config"
	"taskmanager/service/message/internal/repository"
	"taskmanager/service/message/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: message không đọc được lịch sử, API trả danh sách rỗng")
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
	return pool, nil
}

func newMessageReader(pool *pgxpool.Pool) service.MessageReader {
	if pool == nil {
		return repository.EmptyMessageReader{}
	}
	return repository.NewPostgresMessageReader(pool)
}

func newSocialConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(cfg.SocialGRPCDialAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC social (%s): %w", cfg.SocialGRPCDialAddr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newFriendServiceClient(conn *grpc.ClientConn) socialv1.FriendServiceClient {
	return socialv1.NewFriendServiceClient(conn)
}

func newFriendChecker(client socialv1.FriendServiceClient, cfg config.Config) service.FriendChecker {
	return repository.NewSocialClient(client, cfg.UpstreamTimeout)
}

func newBroadcaster(cfg config.Config) service.Broadcaster {
	if !cfg.SoketiConfigured() {
		slog.Warn("thiếu SOKETI_*: tin nhắn không được broadcast realtime")
		return repository.NoopBroadcaster{}
	}
	slog.Info("message broadcast qua soketi", "base_url", cfg.SoketiBaseURL, "channel_prefix", cfg.SoketiChatChannelPrefix)
	return repository.NewSoketiBroadcaster(
		cfg.SoketiBaseURL,
		cfg.SoketiAppID,
		cfg.SoketiAppKey,
		cfg.SoketiAppSecret,
		cfg.SoketiChatChannelPrefix,
		cfg.SoketiTimeout,
	)
}

func newEventPublisher(lc fx.Lifecycle, cfg config.Config) service.EventPublisher {
	writer := repository.NewKafkaWriter(cfg.KafkaBrokers, cfg.KafkaMessageTopic)
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return writer.Close()
		},
	})
	slog.Info("message đẩy sự kiện vào kafka", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaMessageTopic)
	return repository.NewKafkaPublisher(writer)
}

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

	identityv1 "taskmanager/common/gen/go/identity/v1"
	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/social/internal/config"
	"taskmanager/service/social/internal/repository"
	"taskmanager/service/social/internal/service"
)

func newPostgresPool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL trống: social dùng repository in-memory, dữ liệu sẽ mất khi restart")
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
	slog.Info("social dùng repository postgres")
	return pool, nil
}

func newFriendshipRepository(pool *pgxpool.Pool) service.FriendshipRepository {
	if pool == nil {
		return repository.NewInMemoryFriendshipRepository()
	}
	return repository.NewPostgresFriendshipRepository(pool)
}

func dial(lc fx.Lifecycle, name, target string) (*grpc.ClientConn, error) {
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

func newIdentityConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dial(lc, "identity", cfg.IdentityGRPCDialAddr)
}

func newProfileServiceClient(conn *grpc.ClientConn) identityv1.ProfileServiceClient {
	return identityv1.NewProfileServiceClient(conn)
}

func newProfileDirectory(client identityv1.ProfileServiceClient, cfg config.Config) service.ProfileDirectory {
	return repository.NewIdentityDirectory(client, cfg.UpstreamTimeout)
}

func newNotificationConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	return dial(lc, "notification", cfg.NotificationGRPCDialAddr)
}

func newNoticeServiceClient(conn *grpc.ClientConn) notificationv1.NoticeServiceClient {
	return notificationv1.NewNoticeServiceClient(conn)
}

func newFriendNotifier(client notificationv1.NoticeServiceClient, cfg config.Config) service.FriendNotifier {
	return repository.NewNotificationClient(client, cfg.UpstreamTimeout)
}

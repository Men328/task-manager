package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/notification/internal/config"
	"taskmanager/service/notification/internal/handler"
	"taskmanager/service/notification/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newNoticeRepository,
			newNoticePublisher,
			service.NewNoticeService,
			handler.NewNoticeHandler,
		),
		fx.Invoke(setupLogger, warnSoketiConfig, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("notification gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

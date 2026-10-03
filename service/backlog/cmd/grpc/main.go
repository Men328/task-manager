package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/backlog/internal/config"
	"taskmanager/service/backlog/internal/handler"
	"taskmanager/service/backlog/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newBacklogRepository,
			service.NewBacklogService,
			handler.NewBacklogHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("backlog gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

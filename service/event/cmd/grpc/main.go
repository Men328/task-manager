package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/event/internal/config"
	"taskmanager/service/event/internal/handler"
	"taskmanager/service/event/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newEventRepository,
			service.NewEventService,
			handler.NewEventHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("event gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

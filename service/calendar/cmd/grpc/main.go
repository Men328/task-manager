package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/calendar/internal/config"
	"taskmanager/service/calendar/internal/handler"
	"taskmanager/service/calendar/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newScheduleRepository,
			service.NewScheduleService,
			handler.NewScheduleHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("calendar gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/project/internal/config"
	"taskmanager/service/project/internal/handler"
	"taskmanager/service/project/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newProjectRepository,
			service.NewProjectService,
			handler.NewProjectHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("project gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

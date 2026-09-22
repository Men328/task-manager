package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/workspace/internal/config"
	"taskmanager/service/workspace/internal/handler"
	"taskmanager/service/workspace/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newWorkspaceRepository,
			service.NewWorkspaceService,
			handler.NewWorkspaceHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("workspace gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/task/internal/config"
	"taskmanager/service/task/internal/handler"
	"taskmanager/service/task/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newTaskRepository,
			newStatusRepository,
			newTransitionRepository,
			service.NewTaskService,
			service.NewTaskStatusService,
			service.NewStatusTransitionService,
			handler.NewTaskHandler,
			handler.NewTaskStatusHandler,
			handler.NewStatusTransitionHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("task gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

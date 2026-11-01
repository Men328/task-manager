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
			fx.Annotate(newProjectConn, fx.ResultTags(`name:"project"`)),
			fx.Annotate(newProjectServiceClient, fx.ParamTags(`name:"project"`)),
			newProjectChecker,
			fx.Annotate(newNotificationConn, fx.ResultTags(`name:"notification"`)),
			fx.Annotate(newNoticeServiceClient, fx.ParamTags(`name:"notification"`)),
			newReminderWorker,
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
		fx.Invoke(setupLogger, serveGRPC, startReminderWorker),
	)
	if err := app.Err(); err != nil {
		slog.Error("task gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

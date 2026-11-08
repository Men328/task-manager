package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/worker/internal/config"
	"taskmanager/service/worker/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newMessageStore,
			newNotificationConn,
			newTaskNotifier,
			newBroadcaster,
			service.NewMessagePersister,
			service.NewTaskEventHandler,
			fx.Annotate(newChatConsumer, fx.ResultTags(`group:"consumers"`)),
			fx.Annotate(newTaskEventConsumer, fx.ResultTags(`group:"consumers"`)),
			newHealthServer,
		),
		fx.Invoke(
			setupLogger,
			fx.Annotate(runConsumers, fx.ParamTags(``, `group:"consumers"`)),
			serveHealth,
		),
	)
	if err := app.Err(); err != nil {
		slog.Error("worker init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

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
			service.NewMessagePersister,
			newMessageReader,
			newMessageConsumer,
			newHealthServer,
		),
		fx.Invoke(setupLogger, runMessageConsumer, serveHealth),
	)
	if err := app.Err(); err != nil {
		slog.Error("worker init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

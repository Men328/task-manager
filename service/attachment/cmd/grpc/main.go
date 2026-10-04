package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/attachment/internal/config"
	"taskmanager/service/attachment/internal/handler"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newAttachmentRepository,
			newBlobStore,
			newAttachmentService,
			handler.NewAttachmentHandler,
			newListener,
			newGRPCServer,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("attachment gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

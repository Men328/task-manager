package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/message/internal/config"
	"taskmanager/service/message/internal/handler"
	"taskmanager/service/message/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			newSocialConn,
			newFriendServiceClient,
			newFriendChecker,
			newMessageReader,
			newBroadcaster,
			newEventPublisher,
			service.NewMessageService,
			handler.NewMessageHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("message gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

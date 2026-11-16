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
			fx.Annotate(newSocialConn, fx.ResultTags(`name:"social"`)),
			fx.Annotate(newFriendServiceClient, fx.ParamTags(`name:"social"`)),
			fx.Annotate(newTaskConn, fx.ResultTags(`name:"task"`)),
			fx.Annotate(newTaskServiceClient, fx.ParamTags(`name:"task"`)),
			newTaskCreator,
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

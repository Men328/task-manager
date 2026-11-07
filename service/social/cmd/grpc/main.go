package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/social/internal/config"
	"taskmanager/service/social/internal/handler"
	"taskmanager/service/social/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newPostgresPool,
			newListener,
			newGRPCServer,
			fx.Annotate(newIdentityConn, fx.ResultTags(`name:"identity"`)),
			fx.Annotate(newProfileServiceClient, fx.ParamTags(`name:"identity"`)),
			newProfileDirectory,
			fx.Annotate(newNotificationConn, fx.ResultTags(`name:"notification"`)),
			fx.Annotate(newNoticeServiceClient, fx.ParamTags(`name:"notification"`)),
			newFriendNotifier,
			newFriendshipRepository,
			service.NewFriendService,
			handler.NewFriendHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("social gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

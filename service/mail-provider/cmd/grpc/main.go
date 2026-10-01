package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/mail-provider/internal/config"
	"taskmanager/service/mail-provider/internal/handler"
	"taskmanager/service/mail-provider/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newOptions,
			newQueue,
			newListener,
			newGRPCServer,
			newPostgresPool,
			newSubscriptionRepository,
			newGmailClient,
			newTokenRefresher,
			newMailAnalyzer,
			newNotificationSource,
			service.NewTokenManager,
			fx.Annotate(newTaskConn, fx.ResultTags(`name:"task"`)),
			fx.Annotate(newTaskServiceClient, fx.ParamTags(`name:"task"`)),
			newTaskCreator,
			service.NewMailService,
			service.NewWorker,
			fx.Annotate(service.NewPullWorker, fx.ResultTags(`name:"pull"`)),
			handler.NewMailHandler,
		),
		fx.Invoke(
			setupLogger,
			warnMailConfig,
			serveGRPC,
			startWorker,
			fx.Annotate(startPullWorker, fx.ParamTags("", "", `name:"pull"`)),
		),
	)
	if err := app.Err(); err != nil {
		slog.Error("mail-provider gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

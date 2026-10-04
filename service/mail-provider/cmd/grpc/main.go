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
			newBlobStore,
			service.NewTokenManager,
			fx.Annotate(newTaskConn, fx.ResultTags(`name:"task"`)),
			fx.Annotate(newTaskServiceClient, fx.ParamTags(`name:"task"`)),
			newTaskCreator,
			fx.Annotate(newCalendarConn, fx.ResultTags(`name:"calendar"`)),
			fx.Annotate(newCalendarServiceClient, fx.ParamTags(`name:"calendar"`)),
			newScheduleCreator,
			fx.Annotate(newEventConn, fx.ResultTags(`name:"event"`)),
			fx.Annotate(newEventServiceClient, fx.ParamTags(`name:"event"`)),
			newEventCreator,
			fx.Annotate(newBacklogConn, fx.ResultTags(`name:"backlog"`)),
			fx.Annotate(newBacklogServiceClient, fx.ParamTags(`name:"backlog"`)),
			newBacklogCreator,
			fx.Annotate(newNotificationConn, fx.ResultTags(`name:"notification"`)),
			fx.Annotate(newNotificationServiceClient, fx.ParamTags(`name:"notification"`)),
			newNoticePublisher,
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

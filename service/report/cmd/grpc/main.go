package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/report/internal/config"
	"taskmanager/service/report/internal/handler"
	"taskmanager/service/report/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newTaskConn,
			newTaskServiceClient,
			newTaskStatusServiceClient,
			newTaskDataSource,
			fx.Annotate(newEventConn, fx.ResultTags(`name:"event"`)),
			fx.Annotate(newEventServiceClient, fx.ParamTags(`name:"event"`)),
			fx.Annotate(newCalendarConn, fx.ResultTags(`name:"calendar"`)),
			fx.Annotate(newCalendarServiceClient, fx.ParamTags(`name:"calendar"`)),
			fx.Annotate(newBacklogConn, fx.ResultTags(`name:"backlog"`)),
			fx.Annotate(newBacklogServiceClient, fx.ParamTags(`name:"backlog"`)),
			newActivityDataSource,
			newClock,
			newListener,
			newGRPCServer,
			service.NewReportService,
			handler.NewReportHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("report gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

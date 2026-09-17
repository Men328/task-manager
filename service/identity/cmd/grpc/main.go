package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/identity/internal/config"
	"taskmanager/service/identity/internal/handler"
	"taskmanager/service/identity/internal/repository"
	"taskmanager/service/identity/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newListener,
			newGRPCServer,
			fx.Annotate(repository.NewInMemoryProfileRepository, fx.As(new(service.ProfileRepository))),
			service.NewProfileService,
			handler.NewProfileHandler,
		),
		fx.Invoke(setupLogger, serveGRPC),
	)
	if err := app.Err(); err != nil {
		slog.Error("identity gRPC init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

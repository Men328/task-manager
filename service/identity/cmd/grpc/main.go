package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/identity/internal/config"
	"taskmanager/service/identity/internal/handler"
	"taskmanager/service/identity/internal/service"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newListener,
			newGRPCServer,
			newPostgresPool,
			fx.Annotate(newProfileRepository, fx.As(new(service.ProfileRepository))),
			fx.Annotate(newAuthProviderRepository, fx.As(new(service.AuthProviderRepository))),
			service.NewProfileService,
			service.NewAuthService,
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

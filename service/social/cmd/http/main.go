package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/social/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newFriendServiceClient,
			newSessionSigner,
			newFriendRoutes,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, warnSessionConfig, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("social gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/message/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newMessageServiceClient,
			newSessionSigner,
			newMessageRoutes,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, warnSessionConfig, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("message gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

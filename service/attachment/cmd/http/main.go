package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/attachment/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newServeMux,
			newHTTPHandler,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("attachment gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

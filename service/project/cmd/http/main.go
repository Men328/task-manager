package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/project/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("project gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

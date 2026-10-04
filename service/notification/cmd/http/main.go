package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/notification/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newNoticeServiceClient,
			newSessionSigner,
			newNoticeRoutes,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, warnNotificationConfig, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("notification gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

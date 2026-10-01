package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/mail-provider/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newMailConn,
			newMailServiceClient,
			newNotificationVerifier,
			newNotificationRoutes,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, warnHTTPConfig, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("mail-provider gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

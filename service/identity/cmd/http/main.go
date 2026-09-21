package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"

	"taskmanager/service/identity/internal/config"
)

func warnAuthConfig(cfg config.Config) {
	if !cfg.GoogleOAuthConfigured() {
		slog.Warn("thiếu GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET: /v1/auth/google/login sẽ trả IDENTITY_AUTH_NOT_CONFIGURED")
	}
	if !cfg.SessionConfigured() {
		slog.Warn("SESSION_SECRET trống hoặc ngắn hơn 16 ký tự: không phát được session token")
	}
	if cfg.GoogleOAuthConfigured() && cfg.SessionConfigured() {
		slog.Info("đăng nhập Google đã sẵn sàng",
			"redirect_url", cfg.GoogleRedirectURL,
			"frontend_base_url", cfg.FrontendBaseURL,
		)
	}
}

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newGRPCConn,
			newProfileServiceClient,
			newSessionSigner,
			newGoogleOAuth,
			newAuthRoutes,
			newServeMux,
			newHTTPListener,
			newHTTPServer,
		),
		fx.Invoke(setupLogger, warnAuthConfig, serveHTTP),
	)
	if err := app.Err(); err != nil {
		slog.Error("identity gateway init failed", "error", err)
		os.Exit(1)
	}
	app.Run()
}

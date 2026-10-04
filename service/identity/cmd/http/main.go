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
	if cfg.SoketiConfigured() {
		slog.Info("uỷ quyền kênh soketi đã sẵn sàng", "channel_prefix", cfg.SoketiChannelPrefix)
	} else {
		slog.Warn("thiếu SOKETI_APP_KEY/SOKETI_APP_SECRET: /v1/auth/soketi sẽ trả IDENTITY_AUTH_SOKETI_NOT_CONFIGURED")
	}
}

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			fx.Annotate(newGRPCConn, fx.ResultTags(`name:"identity"`)),
			fx.Annotate(newMailConn, fx.ResultTags(`name:"mail"`)),
			fx.Annotate(newProfileServiceClient, fx.ParamTags(`name:"identity"`)),
			fx.Annotate(newMailServiceClient, fx.ParamTags(`name:"mail"`)),
			newSessionSigner,
			newGoogleOAuth,
			newSoketiAuth,
			newAuthRoutes,
			fx.Annotate(newServeMux, fx.ParamTags("", `name:"identity"`, "")),
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

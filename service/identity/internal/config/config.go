package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceName string

	GRPCAddr string

	GRPCDialAddr string
	HTTPAddr     string
	LogLevel     string

	DatabaseURL string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	FrontendBaseURL string

	SessionSecret string
	SessionTTL    time.Duration
	CookieSecure  bool
}

func Load() Config {
	return Config{
		ServiceName:  getenv("SERVICE_NAME", "identity"),
		GRPCAddr:     getenv("GRPC_ADDR", ":9081"),
		GRPCDialAddr: getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:     getenv("HTTP_ADDR", ":8081"),
		LogLevel:     getenv("LOG_LEVEL", "info"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getenv("GOOGLE_REDIRECT_URL", "http://localhost:8081/v1/auth/google/callback"),

		FrontendBaseURL: strings.TrimRight(getenv("FRONTEND_BASE_URL", "http://localhost:5173"), "/"),

		SessionSecret: os.Getenv("SESSION_SECRET"),
		SessionTTL:    getduration("SESSION_TTL", 720*time.Hour),
		CookieSecure:  getbool("COOKIE_SECURE", false),
	}
}

func (c Config) GoogleOAuthConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func (c Config) SessionConfigured() bool {
	return len(c.SessionSecret) >= 16
}

func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c Config) GRPCDialTarget() string {
	if c.GRPCDialAddr != "" {
		return c.GRPCDialAddr
	}
	if strings.HasPrefix(c.GRPCAddr, ":") {
		return "127.0.0.1" + c.GRPCAddr
	}
	return c.GRPCAddr
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getbool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getduration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

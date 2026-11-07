package config

import (
	"log/slog"
	"os"
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

	// SessionSecret dùng để xác thực session JWT do identity phát hành.
	SessionSecret string

	IdentityGRPCDialAddr     string
	NotificationGRPCDialAddr string
	UpstreamTimeout          time.Duration
}

func Load() Config {
	return Config{
		ServiceName:              getenv("SERVICE_NAME", "social"),
		GRPCAddr:                 getenv("GRPC_ADDR", ":9091"),
		GRPCDialAddr:             getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:                 getenv("HTTP_ADDR", ":8091"),
		LogLevel:                 getenv("LOG_LEVEL", "info"),
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		SessionSecret:            os.Getenv("SESSION_SECRET"),
		IdentityGRPCDialAddr:     getenv("IDENTITY_GRPC_DIAL_ADDR", "localhost:9081"),
		NotificationGRPCDialAddr: getenv("NOTIFICATION_GRPC_DIAL_ADDR", "localhost:9088"),
		UpstreamTimeout:          getduration("UPSTREAM_TIMEOUT", 10*time.Second),
	}
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

func getduration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}

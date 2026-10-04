package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServiceName string

	GRPCAddr     string
	GRPCDialAddr string
	HTTPAddr     string
	LogLevel     string

	DatabaseURL string

	SoketiBaseURL       string
	SoketiAppID         string
	SoketiAppKey        string
	SoketiAppSecret     string
	SoketiChannelPrefix string
	SoketiTimeout       time.Duration

	SessionSecret string
}

func Load() Config {
	return Config{
		ServiceName:  getenv("SERVICE_NAME", "notification"),
		GRPCAddr:     getenv("GRPC_ADDR", ":9088"),
		GRPCDialAddr: getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:     getenv("HTTP_ADDR", ":8088"),
		LogLevel:     getenv("LOG_LEVEL", "info"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		SoketiBaseURL:       getenv("SOKETI_BASE_URL", "http://soketi:6001"),
		SoketiAppID:         getenv("SOKETI_APP_ID", "task-manager"),
		SoketiAppKey:        os.Getenv("SOKETI_APP_KEY"),
		SoketiAppSecret:     os.Getenv("SOKETI_APP_SECRET"),
		SoketiChannelPrefix: getenv("SOKETI_CHANNEL_PREFIX", "noti-internal-"),
		SoketiTimeout:       getduration("SOKETI_TIMEOUT", 10*time.Second),

		SessionSecret: os.Getenv("SESSION_SECRET"),
	}
}

func (c Config) SoketiConfigured() bool {
	return strings.TrimSpace(c.SoketiBaseURL) != "" &&
		strings.TrimSpace(c.SoketiAppID) != "" &&
		strings.TrimSpace(c.SoketiAppKey) != "" &&
		strings.TrimSpace(c.SoketiAppSecret) != ""
}

func (c Config) SessionConfigured() bool {
	return len(c.SessionSecret) >= 16
}

func (c Config) ChannelPrefix() string {
	prefix := strings.TrimSpace(c.SoketiChannelPrefix)
	if prefix == "" {
		return "noti-internal-"
	}
	return prefix
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

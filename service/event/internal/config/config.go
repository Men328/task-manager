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

	ProjectGRPCDialAddr string
	ProjectTimeout      time.Duration
}

func Load() Config {
	return Config{
		ServiceName:  getenv("SERVICE_NAME", "event"),
		GRPCAddr:     getenv("GRPC_ADDR", ":9085"),
		GRPCDialAddr: getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:     getenv("HTTP_ADDR", ":8085"),
		LogLevel:     getenv("LOG_LEVEL", "info"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),

		ProjectGRPCDialAddr: os.Getenv("PROJECT_GRPC_DIAL_ADDR"),
		ProjectTimeout:      getduration("PROJECT_TIMEOUT", 10*time.Second),
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

func (c Config) ProjectDialTarget() string {
	if c.ProjectGRPCDialAddr != "" {
		return c.ProjectGRPCDialAddr
	}
	return "127.0.0.1:9090"
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

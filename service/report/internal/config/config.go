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

	TaskGRPCDialAddr string
	TaskTimeout      time.Duration

	EventGRPCDialAddr    string
	CalendarGRPCDialAddr string
	BacklogGRPCDialAddr  string
	ActivityTimeout      time.Duration
}

func Load() Config {
	return Config{
		ServiceName:          getenv("SERVICE_NAME", "report"),
		GRPCAddr:             getenv("GRPC_ADDR", ":9087"),
		GRPCDialAddr:         getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:             getenv("HTTP_ADDR", ":8087"),
		LogLevel:             getenv("LOG_LEVEL", "info"),
		TaskGRPCDialAddr:     os.Getenv("TASK_GRPC_DIAL_ADDR"),
		TaskTimeout:          getDuration("TASK_TIMEOUT", 10*time.Second),
		EventGRPCDialAddr:    os.Getenv("EVENT_GRPC_DIAL_ADDR"),
		CalendarGRPCDialAddr: os.Getenv("CALENDAR_GRPC_DIAL_ADDR"),
		BacklogGRPCDialAddr:  os.Getenv("BACKLOG_GRPC_DIAL_ADDR"),
		ActivityTimeout:      getDuration("ACTIVITY_TIMEOUT", 10*time.Second),
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

func (c Config) TaskDialTarget() string {
	if c.TaskGRPCDialAddr != "" {
		return c.TaskGRPCDialAddr
	}
	return "127.0.0.1:9082"
}

func (c Config) EventDialTarget() string {
	if c.EventGRPCDialAddr != "" {
		return c.EventGRPCDialAddr
	}
	return "127.0.0.1:9085"
}

func (c Config) CalendarDialTarget() string {
	if c.CalendarGRPCDialAddr != "" {
		return c.CalendarGRPCDialAddr
	}
	return "127.0.0.1:9083"
}

func (c Config) BacklogDialTarget() string {
	if c.BacklogGRPCDialAddr != "" {
		return c.BacklogGRPCDialAddr
	}
	return "127.0.0.1:9086"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

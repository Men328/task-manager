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

	ProjectGRPCDialAddr string
	ProjectTimeout      time.Duration

	NotificationGRPCDialAddr string
	NotificationTimeout      time.Duration

	ReminderEnabled  bool
	ReminderInterval time.Duration
	ReminderBatch    int
	ReminderMaxLate  time.Duration
	ReminderTimezone string
}

func Load() Config {
	return Config{
		ServiceName:  getenv("SERVICE_NAME", "task"),
		GRPCAddr:     getenv("GRPC_ADDR", ":9082"),
		GRPCDialAddr: getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:     getenv("HTTP_ADDR", ":8082"),
		LogLevel:     getenv("LOG_LEVEL", "info"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),

		ProjectGRPCDialAddr: os.Getenv("PROJECT_GRPC_DIAL_ADDR"),
		ProjectTimeout:      getduration("PROJECT_TIMEOUT", 10*time.Second),

		NotificationGRPCDialAddr: os.Getenv("NOTIFICATION_GRPC_DIAL_ADDR"),
		NotificationTimeout:      getduration("NOTIFICATION_TIMEOUT", 10*time.Second),

		ReminderEnabled:  getbool("REMINDER_ENABLED", true),
		ReminderInterval: getduration("REMINDER_INTERVAL", 30*time.Second),
		ReminderBatch:    getint("REMINDER_BATCH", 100),
		ReminderMaxLate:  getduration("REMINDER_MAX_LATE", 6*time.Hour),
		ReminderTimezone: getenv("REMINDER_TIMEZONE", "Asia/Ho_Chi_Minh"),
	}
}

func (c Config) NotificationDialTarget() string {
	if c.NotificationGRPCDialAddr != "" {
		return c.NotificationGRPCDialAddr
	}
	return "127.0.0.1:9088"
}

func (c Config) ReminderLocation() *time.Location {
	loc, err := time.LoadLocation(c.ReminderTimezone)
	if err != nil {
		slog.Warn("REMINDER_TIMEZONE không hợp lệ, dùng UTC", "value", c.ReminderTimezone, "error", err)
		return time.UTC
	}
	return loc
}

func getbool(key string, fallback bool) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return v
}

func getint(key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
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

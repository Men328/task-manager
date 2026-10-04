package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultMaxAttachmentBytes = 10 << 20

type Config struct {
	ServiceName string

	GRPCAddr string

	GRPCDialAddr string
	HTTPAddr     string
	LogLevel     string

	DatabaseURL string

	MaxAttachmentBytes int64

	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3Region    string
	S3UseSSL    bool
	S3Timeout   time.Duration
}

func Load() Config {
	return Config{
		ServiceName:        getenv("SERVICE_NAME", "attachment"),
		GRPCAddr:           getenv("GRPC_ADDR", ":9089"),
		GRPCDialAddr:       getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:           getenv("HTTP_ADDR", ":8089"),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		MaxAttachmentBytes: getint64("ATTACHMENT_MAX_BYTES", defaultMaxAttachmentBytes),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		S3AccessKey:        getenv("S3_ACCESS_KEY", ""),
		S3SecretKey:        getenv("S3_SECRET_KEY", ""),
		S3Bucket:           getenv("S3_BUCKET", "task-manager"),
		S3Region:           getenv("S3_REGION", "us-east-1"),
		S3UseSSL:           getbool("S3_USE_SSL", false),
		S3Timeout:          getduration("S3_TIMEOUT", 30*time.Second),
	}
}

func (c Config) StorageConfigured() bool {
	return strings.TrimSpace(c.S3Endpoint) != ""
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

func getint64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func getbool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getduration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

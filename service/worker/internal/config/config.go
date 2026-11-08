package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	HealthAddr  string
	LogLevel    string

	DatabaseURL string

	KafkaBrokers        []string
	KafkaMessageTopic   string
	KafkaMessageGroupID string
	RetryInitialBackoff time.Duration
	RetryMaxBackoff     time.Duration
	PersistTimeout      time.Duration
}

func Load() Config {
	return Config{
		ServiceName:         getenv("SERVICE_NAME", "worker"),
		HealthAddr:          getenv("HEALTH_ADDR", ":8094"),
		LogLevel:            getenv("LOG_LEVEL", "info"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		KafkaBrokers:        splitList(getenv("KAFKA_BROKERS", "localhost:29092")),
		KafkaMessageTopic:   getenv("KAFKA_MESSAGE_TOPIC", "chat.messages"),
		KafkaMessageGroupID: getenv("KAFKA_MESSAGE_GROUP_ID", "message-worker"),
		RetryInitialBackoff: getduration("RETRY_INITIAL_BACKOFF", 500*time.Millisecond),
		RetryMaxBackoff:     getduration("RETRY_MAX_BACKOFF", 30*time.Second),
		PersistTimeout:      getduration("PERSIST_TIMEOUT", 10*time.Second),
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getduration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

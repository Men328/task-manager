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

	SessionSecret string

	SocialGRPCDialAddr string
	UpstreamTimeout    time.Duration

	SoketiBaseURL           string
	SoketiAppID             string
	SoketiAppKey            string
	SoketiAppSecret         string
	SoketiChatChannelPrefix string
	SoketiTimeout           time.Duration

	KafkaBrokers      []string
	KafkaMessageTopic string
}

func Load() Config {
	return Config{
		ServiceName:             getenv("SERVICE_NAME", "message"),
		GRPCAddr:                getenv("GRPC_ADDR", ":9092"),
		GRPCDialAddr:            getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:                getenv("HTTP_ADDR", ":8092"),
		LogLevel:                getenv("LOG_LEVEL", "info"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		SessionSecret:           os.Getenv("SESSION_SECRET"),
		SocialGRPCDialAddr:      getenv("SOCIAL_GRPC_DIAL_ADDR", "localhost:9091"),
		UpstreamTimeout:         getduration("UPSTREAM_TIMEOUT", 5*time.Second),
		SoketiBaseURL:           getenv("SOKETI_BASE_URL", "http://localhost:6001"),
		SoketiAppID:             os.Getenv("SOKETI_APP_ID"),
		SoketiAppKey:            os.Getenv("SOKETI_APP_KEY"),
		SoketiAppSecret:         os.Getenv("SOKETI_APP_SECRET"),
		SoketiChatChannelPrefix: getenv("SOKETI_CHAT_CHANNEL_PREFIX", "chat-"),
		SoketiTimeout:           getduration("SOKETI_TIMEOUT", 3*time.Second),
		KafkaBrokers:            splitList(getenv("KAFKA_BROKERS", "localhost:29092")),
		KafkaMessageTopic:       getenv("KAFKA_MESSAGE_TOPIC", "chat.messages"),
	}
}

func (c Config) SessionConfigured() bool {
	return len(c.SessionSecret) >= 16
}

func (c Config) SoketiConfigured() bool {
	return strings.TrimSpace(c.SoketiBaseURL) != "" &&
		strings.TrimSpace(c.SoketiAppID) != "" &&
		strings.TrimSpace(c.SoketiAppKey) != "" &&
		strings.TrimSpace(c.SoketiAppSecret) != ""
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

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

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

	PubSubTopic          string
	PubSubAudience       string
	PubSubServiceAccount string
	WatchLabelIDs        []string
	WatchRenewInterval   time.Duration
	WatchRenewThreshold  time.Duration
	NotificationPath     string

	PubSubBaseURL    string
	PullSubscription string
	PullMaxMessages  int
	PullTimeout      time.Duration
	PullRetryDelay   time.Duration
	PullHeartbeat    time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleTokenURL     string

	GmailBaseURL      string
	GmailTimeout      time.Duration
	GmailMaxBodyBytes int

	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string
	DeepSeekTimeout time.Duration

	TaskGRPCDialAddr      string
	WorkspaceGRPCDialAddr string
	DefaultPriority       string
	TaskTimeout           time.Duration

	QueueSize   int
	WorkerCount int
}

func Load() Config {
	return Config{
		ServiceName:  getenv("SERVICE_NAME", "mail-provider"),
		GRPCAddr:     getenv("GRPC_ADDR", ":9084"),
		GRPCDialAddr: getenv("GRPC_DIAL_ADDR", ""),
		HTTPAddr:     getenv("HTTP_ADDR", ":8084"),
		LogLevel:     getenv("LOG_LEVEL", "info"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		PubSubTopic:          os.Getenv("MAIL_PUBSUB_TOPIC"),
		PubSubAudience:       os.Getenv("MAIL_PUBSUB_AUDIENCE"),
		PubSubServiceAccount: os.Getenv("MAIL_PUBSUB_SERVICE_ACCOUNT"),
		WatchLabelIDs:        getcsv("MAIL_WATCH_LABEL_IDS", []string{"INBOX"}),
		WatchRenewInterval:   getduration("MAIL_WATCH_RENEW_INTERVAL", 12*time.Hour),
		WatchRenewThreshold:  getduration("MAIL_WATCH_RENEW_THRESHOLD", 24*time.Hour),
		NotificationPath:     getenv("MAIL_NOTIFICATION_PATH", "/v1/notifications"),

		PubSubBaseURL:    getenv("MAIL_PUBSUB_BASE_URL", "https://pubsub.googleapis.com"),
		PullSubscription: os.Getenv("MAIL_PULL_SUBSCRIPTION"),
		PullMaxMessages:  getint("MAIL_PULL_MAX_MESSAGES", 10),
		PullTimeout:      getduration("MAIL_PULL_TIMEOUT", 2*time.Minute),
		PullRetryDelay:   getduration("MAIL_PULL_RETRY_DELAY", 5*time.Second),
		PullHeartbeat:    getdurationZero("MAIL_PULL_HEARTBEAT", 5*time.Minute),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleTokenURL:     getenv("GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token"),

		GmailBaseURL:      getenv("GMAIL_BASE_URL", "https://gmail.googleapis.com"),
		GmailTimeout:      getduration("GMAIL_TIMEOUT", 20*time.Second),
		GmailMaxBodyBytes: getint("GMAIL_MAX_BODY_BYTES", 32*1024),

		DeepSeekAPIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: strings.TrimRight(getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"), "/"),
		DeepSeekModel:   getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekTimeout: getduration("DEEPSEEK_TIMEOUT", 60*time.Second),

		TaskGRPCDialAddr:      getenv("TASK_GRPC_DIAL_ADDR", ""),
		WorkspaceGRPCDialAddr: getenv("WORKSPACE_GRPC_DIAL_ADDR", ""),
		DefaultPriority:       getenv("MAIL_DEFAULT_PRIORITY", "medium"),
		TaskTimeout:           getduration("MAIL_TASK_TIMEOUT", 15*time.Second),

		QueueSize:   getint("MAIL_QUEUE_SIZE", 256),
		WorkerCount: getint("MAIL_WORKER_COUNT", 2),
	}
}

func (c Config) PubSubConfigured() bool {
	return c.PubSubTopic != ""
}

func (c Config) PullConfigured() bool {
	return strings.TrimSpace(c.PullSubscription) != ""
}

func (c Config) AnalyzerConfigured() bool {
	return c.DeepSeekAPIKey != ""
}

func (c Config) TokenRefreshConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
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
	return dialTarget(c.TaskGRPCDialAddr, ":9082")
}

func (c Config) WorkspaceDialTarget() string {
	return dialTarget(c.WorkspaceGRPCDialAddr, ":9083")
}

func dialTarget(configured string, fallbackPort string) string {
	if configured != "" {
		return configured
	}
	return "127.0.0.1" + fallbackPort
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getint(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
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

func getdurationZero(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func getcsv(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

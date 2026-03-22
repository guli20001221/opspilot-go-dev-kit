package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultEnv                       = "development"
	defaultLogLevel                  = "INFO"
	defaultAPIListenAddr             = ":8080"
	defaultPostgresDSN               = "postgres://opspilot:opspilot@localhost:5432/opspilot?sslmode=disable"
	defaultTemporalEnabled           = false
	defaultTemporalAddress           = "localhost:7233"
	defaultTemporalNamespace         = "default"
	defaultTemporalTaskQueue         = "opspilot-report-tasks"
	defaultApprovedToolFailOnApprove = false
	defaultWorkerPollInterval        = 1 * time.Second
	defaultWorkerShutdownTimeout     = 10 * time.Second
)

// Config holds the minimum process configuration required by the foundation slice.
type Config struct {
	Env                       string
	LogLevel                  string
	APIListenAddr             string
	PostgresDSN               string
	TemporalEnabled           bool
	TemporalAddress           string
	TemporalNamespace         string
	TemporalTaskQueue         string
	TicketAPIBaseURL          string
	TicketAPIToken            string
	ApprovedToolFailOnApprove bool
	WorkerPollInterval        time.Duration
	WorkerShutdownTimeout     time.Duration
}

// Load reads process configuration from environment variables and applies safe defaults.
func Load() (Config, error) {
	cfg := Config{
		Env:                       getEnv("OPSPILOT_ENV", defaultEnv),
		LogLevel:                  getEnv("OPSPILOT_LOG_LEVEL", defaultLogLevel),
		APIListenAddr:             getEnv("OPSPILOT_API_LISTEN_ADDR", defaultAPIListenAddr),
		PostgresDSN:               getEnv("OPSPILOT_POSTGRES_DSN", defaultPostgresDSN),
		TemporalEnabled:           defaultTemporalEnabled,
		TemporalAddress:           getEnv("OPSPILOT_TEMPORAL_ADDRESS", defaultTemporalAddress),
		TemporalNamespace:         getEnv("OPSPILOT_TEMPORAL_NAMESPACE", defaultTemporalNamespace),
		TemporalTaskQueue:         getEnv("OPSPILOT_TEMPORAL_TASK_QUEUE", defaultTemporalTaskQueue),
		TicketAPIBaseURL:          getEnv("OPSPILOT_TICKET_API_BASE_URL", ""),
		TicketAPIToken:            getEnv("OPSPILOT_TICKET_API_TOKEN", ""),
		ApprovedToolFailOnApprove: defaultApprovedToolFailOnApprove,
		WorkerPollInterval:        defaultWorkerPollInterval,
		WorkerShutdownTimeout:     defaultWorkerShutdownTimeout,
	}

	if raw := os.Getenv("OPSPILOT_TEMPORAL_ENABLED"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_TEMPORAL_ENABLED: %w", err)
		}
		cfg.TemporalEnabled = enabled
	}
	if raw := os.Getenv("OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE: %w", err)
		}
		cfg.ApprovedToolFailOnApprove = enabled
	}

	if raw := os.Getenv("OPSPILOT_WORKER_POLL_INTERVAL"); raw != "" {
		interval, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_WORKER_POLL_INTERVAL: %w", err)
		}
		cfg.WorkerPollInterval = interval
	}

	if raw := os.Getenv("OPSPILOT_WORKER_SHUTDOWN_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_WORKER_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.WorkerShutdownTimeout = timeout
	}

	if cfg.APIListenAddr == "" {
		return Config{}, fmt.Errorf("OPSPILOT_API_LISTEN_ADDR must not be empty")
	}
	if cfg.PostgresDSN == "" {
		return Config{}, fmt.Errorf("OPSPILOT_POSTGRES_DSN must not be empty")
	}
	if cfg.TemporalAddress == "" {
		return Config{}, fmt.Errorf("OPSPILOT_TEMPORAL_ADDRESS must not be empty")
	}
	if cfg.TemporalNamespace == "" {
		return Config{}, fmt.Errorf("OPSPILOT_TEMPORAL_NAMESPACE must not be empty")
	}
	if cfg.TemporalTaskQueue == "" {
		return Config{}, fmt.Errorf("OPSPILOT_TEMPORAL_TASK_QUEUE must not be empty")
	}
	if cfg.WorkerPollInterval <= 0 {
		return Config{}, fmt.Errorf("OPSPILOT_WORKER_POLL_INTERVAL must be positive")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

package config

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultEnv                   = "development"
	defaultLogLevel              = "INFO"
	defaultAPIListenAddr         = ":8080"
	defaultPostgresDSN           = "postgres://opspilot:opspilot@localhost:5432/opspilot?sslmode=disable"
	defaultWorkerPollInterval    = 1 * time.Second
	defaultWorkerShutdownTimeout = 10 * time.Second
)

// Config holds the minimum process configuration required by the foundation slice.
type Config struct {
	Env                   string
	LogLevel              string
	APIListenAddr         string
	PostgresDSN           string
	WorkerPollInterval    time.Duration
	WorkerShutdownTimeout time.Duration
}

// Load reads process configuration from environment variables and applies safe defaults.
func Load() (Config, error) {
	cfg := Config{
		Env:                   getEnv("OPSPILOT_ENV", defaultEnv),
		LogLevel:              getEnv("OPSPILOT_LOG_LEVEL", defaultLogLevel),
		APIListenAddr:         getEnv("OPSPILOT_API_LISTEN_ADDR", defaultAPIListenAddr),
		PostgresDSN:           getEnv("OPSPILOT_POSTGRES_DSN", defaultPostgresDSN),
		WorkerPollInterval:    defaultWorkerPollInterval,
		WorkerShutdownTimeout: defaultWorkerShutdownTimeout,
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

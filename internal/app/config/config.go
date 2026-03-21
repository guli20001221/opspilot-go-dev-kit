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
	defaultWorkerShutdownTimeout = 10 * time.Second
)

// Config holds the minimum process configuration required by the foundation slice.
type Config struct {
	Env                   string
	LogLevel              string
	APIListenAddr         string
	WorkerShutdownTimeout time.Duration
}

// Load reads process configuration from environment variables and applies safe defaults.
func Load() (Config, error) {
	cfg := Config{
		Env:                   getEnv("OPSPILOT_ENV", defaultEnv),
		LogLevel:              getEnv("OPSPILOT_LOG_LEVEL", defaultLogLevel),
		APIListenAddr:         getEnv("OPSPILOT_API_LISTEN_ADDR", defaultAPIListenAddr),
		WorkerShutdownTimeout: defaultWorkerShutdownTimeout,
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

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("OPSPILOT_ENV", "")
	t.Setenv("OPSPILOT_LOG_LEVEL", "")
	t.Setenv("OPSPILOT_API_LISTEN_ADDR", "")
	t.Setenv("OPSPILOT_POSTGRES_DSN", "")
	t.Setenv("OPSPILOT_WORKER_SHUTDOWN_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "development" {
		t.Fatalf("Env = %q, want %q", cfg.Env, "development")
	}
	if cfg.LogLevel != "INFO" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "INFO")
	}
	if cfg.APIListenAddr != ":8080" {
		t.Fatalf("APIListenAddr = %q, want %q", cfg.APIListenAddr, ":8080")
	}
	if cfg.PostgresDSN != "postgres://opspilot:opspilot@localhost:5432/opspilot?sslmode=disable" {
		t.Fatalf("PostgresDSN = %q, want %q", cfg.PostgresDSN, "postgres://opspilot:opspilot@localhost:5432/opspilot?sslmode=disable")
	}
	if cfg.WorkerShutdownTimeout != 10*time.Second {
		t.Fatalf("WorkerShutdownTimeout = %s, want %s", cfg.WorkerShutdownTimeout, 10*time.Second)
	}
}

func TestLoadUsesEnvOverrides(t *testing.T) {
	t.Setenv("OPSPILOT_ENV", "production")
	t.Setenv("OPSPILOT_LOG_LEVEL", "DEBUG")
	t.Setenv("OPSPILOT_API_LISTEN_ADDR", ":18080")
	t.Setenv("OPSPILOT_POSTGRES_DSN", "postgres://custom")
	t.Setenv("OPSPILOT_WORKER_SHUTDOWN_TIMEOUT", "25s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "production" {
		t.Fatalf("Env = %q, want %q", cfg.Env, "production")
	}
	if cfg.LogLevel != "DEBUG" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "DEBUG")
	}
	if cfg.APIListenAddr != ":18080" {
		t.Fatalf("APIListenAddr = %q, want %q", cfg.APIListenAddr, ":18080")
	}
	if cfg.PostgresDSN != "postgres://custom" {
		t.Fatalf("PostgresDSN = %q, want %q", cfg.PostgresDSN, "postgres://custom")
	}
	if cfg.WorkerShutdownTimeout != 25*time.Second {
		t.Fatalf("WorkerShutdownTimeout = %s, want %s", cfg.WorkerShutdownTimeout, 25*time.Second)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("OPSPILOT_WORKER_SHUTDOWN_TIMEOUT", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

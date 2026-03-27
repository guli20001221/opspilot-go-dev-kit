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
	t.Setenv("OPSPILOT_EVAL_JUDGE_PROVIDER", "")
	t.Setenv("OPSPILOT_EVAL_JUDGE_BASE_URL", "")
	t.Setenv("OPSPILOT_EVAL_JUDGE_API_KEY", "")
	t.Setenv("OPSPILOT_EVAL_JUDGE_MODEL", "")
	t.Setenv("OPSPILOT_EVAL_JUDGE_TIMEOUT", "")
	t.Setenv("OPSPILOT_WORKER_POLL_INTERVAL", "")
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
	if cfg.EvalJudgeProvider != "placeholder" {
		t.Fatalf("EvalJudgeProvider = %q, want %q", cfg.EvalJudgeProvider, "placeholder")
	}
	if cfg.EvalJudgeBaseURL != "" {
		t.Fatalf("EvalJudgeBaseURL = %q, want empty", cfg.EvalJudgeBaseURL)
	}
	if cfg.EvalJudgeAPIKey != "" {
		t.Fatalf("EvalJudgeAPIKey = %q, want empty", cfg.EvalJudgeAPIKey)
	}
	if cfg.EvalJudgeModel != "" {
		t.Fatalf("EvalJudgeModel = %q, want empty", cfg.EvalJudgeModel)
	}
	if cfg.EvalJudgeTimeout != 15*time.Second {
		t.Fatalf("EvalJudgeTimeout = %s, want %s", cfg.EvalJudgeTimeout, 15*time.Second)
	}
	if cfg.TemporalEnabled {
		t.Fatal("TemporalEnabled = true, want false")
	}
	if cfg.TemporalAddress != "localhost:7233" {
		t.Fatalf("TemporalAddress = %q, want %q", cfg.TemporalAddress, "localhost:7233")
	}
	if cfg.TemporalNamespace != "default" {
		t.Fatalf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, "default")
	}
	if cfg.TemporalTaskQueue != "opspilot-report-tasks" {
		t.Fatalf("TemporalTaskQueue = %q, want %q", cfg.TemporalTaskQueue, "opspilot-report-tasks")
	}
	if cfg.ApprovedToolFailOnApprove {
		t.Fatal("ApprovedToolFailOnApprove = true, want false")
	}
	if cfg.EvalRunFailAll {
		t.Fatal("EvalRunFailAll = true, want false")
	}
	if cfg.WorkerPollInterval != 1*time.Second {
		t.Fatalf("WorkerPollInterval = %s, want %s", cfg.WorkerPollInterval, 1*time.Second)
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
	t.Setenv("OPSPILOT_EVAL_JUDGE_PROVIDER", "http_json")
	t.Setenv("OPSPILOT_EVAL_JUDGE_BASE_URL", "http://judge.internal")
	t.Setenv("OPSPILOT_EVAL_JUDGE_API_KEY", "judge-token")
	t.Setenv("OPSPILOT_EVAL_JUDGE_MODEL", "judge-demo")
	t.Setenv("OPSPILOT_EVAL_JUDGE_TIMEOUT", "12s")
	t.Setenv("OPSPILOT_TEMPORAL_ENABLED", "true")
	t.Setenv("OPSPILOT_TEMPORAL_ADDRESS", "temporal:7233")
	t.Setenv("OPSPILOT_TEMPORAL_NAMESPACE", "opspilot")
	t.Setenv("OPSPILOT_TEMPORAL_TASK_QUEUE", "opspilot-runtime")
	t.Setenv("OPSPILOT_TICKET_API_BASE_URL", "http://tickets.internal")
	t.Setenv("OPSPILOT_TICKET_API_TOKEN", "secret-token")
	t.Setenv("OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE", "true")
	t.Setenv("OPSPILOT_EVAL_RUN_FAIL_ALL", "true")
	t.Setenv("OPSPILOT_WORKER_POLL_INTERVAL", "3s")
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
	if cfg.EvalJudgeProvider != "http_json" {
		t.Fatalf("EvalJudgeProvider = %q, want %q", cfg.EvalJudgeProvider, "http_json")
	}
	if cfg.EvalJudgeBaseURL != "http://judge.internal" {
		t.Fatalf("EvalJudgeBaseURL = %q, want %q", cfg.EvalJudgeBaseURL, "http://judge.internal")
	}
	if cfg.EvalJudgeAPIKey != "judge-token" {
		t.Fatalf("EvalJudgeAPIKey = %q, want %q", cfg.EvalJudgeAPIKey, "judge-token")
	}
	if cfg.EvalJudgeModel != "judge-demo" {
		t.Fatalf("EvalJudgeModel = %q, want %q", cfg.EvalJudgeModel, "judge-demo")
	}
	if cfg.EvalJudgeTimeout != 12*time.Second {
		t.Fatalf("EvalJudgeTimeout = %s, want %s", cfg.EvalJudgeTimeout, 12*time.Second)
	}
	if !cfg.TemporalEnabled {
		t.Fatal("TemporalEnabled = false, want true")
	}
	if cfg.TemporalAddress != "temporal:7233" {
		t.Fatalf("TemporalAddress = %q, want %q", cfg.TemporalAddress, "temporal:7233")
	}
	if cfg.TemporalNamespace != "opspilot" {
		t.Fatalf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, "opspilot")
	}
	if cfg.TemporalTaskQueue != "opspilot-runtime" {
		t.Fatalf("TemporalTaskQueue = %q, want %q", cfg.TemporalTaskQueue, "opspilot-runtime")
	}
	if cfg.TicketAPIBaseURL != "http://tickets.internal" {
		t.Fatalf("TicketAPIBaseURL = %q, want %q", cfg.TicketAPIBaseURL, "http://tickets.internal")
	}
	if cfg.TicketAPIToken != "secret-token" {
		t.Fatalf("TicketAPIToken = %q, want %q", cfg.TicketAPIToken, "secret-token")
	}
	if !cfg.ApprovedToolFailOnApprove {
		t.Fatal("ApprovedToolFailOnApprove = false, want true")
	}
	if !cfg.EvalRunFailAll {
		t.Fatal("EvalRunFailAll = false, want true")
	}
	if cfg.WorkerPollInterval != 3*time.Second {
		t.Fatalf("WorkerPollInterval = %s, want %s", cfg.WorkerPollInterval, 3*time.Second)
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

func TestLoadRejectsInvalidPollInterval(t *testing.T) {
	t.Setenv("OPSPILOT_WORKER_POLL_INTERVAL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidEvalJudgeTimeout(t *testing.T) {
	t.Setenv("OPSPILOT_EVAL_JUDGE_TIMEOUT", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidTemporalEnabled(t *testing.T) {
	t.Setenv("OPSPILOT_TEMPORAL_ENABLED", "not-a-bool")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidApprovedToolFailOnApprove(t *testing.T) {
	t.Setenv("OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE", "not-a-bool")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidEvalRunFailAll(t *testing.T) {
	t.Setenv("OPSPILOT_EVAL_RUN_FAIL_ALL", "not-a-bool")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

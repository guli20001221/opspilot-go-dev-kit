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
	defaultEvalJudgeProvider         = "placeholder"
	defaultTemporalEnabled           = false
	defaultTemporalAddress           = "localhost:7233"
	defaultTemporalNamespace         = "default"
	defaultTemporalTaskQueue         = "opspilot-report-tasks"
	defaultApprovedToolFailOnApprove = false
	defaultEvalRunFailAll            = false
	defaultEvalJudgeTimeout          = 15 * time.Second
	defaultWorkerPollInterval        = 1 * time.Second
	defaultWorkerShutdownTimeout     = 10 * time.Second
	defaultLLMProvider               = "placeholder"
	defaultLLMTimeout                = 30 * time.Second
	defaultEmbeddingProvider         = "placeholder"
	defaultEmbeddingTimeout          = 15 * time.Second
)

// Config holds the minimum process configuration required by the foundation slice.
type Config struct {
	Env                       string
	LogLevel                  string
	APIListenAddr             string
	PostgresDSN               string
	EvalJudgeProvider         string
	EvalJudgeBaseURL          string
	EvalJudgeAPIKey           string
	EvalJudgeModel            string
	EvalJudgeTimeout          time.Duration
	TemporalEnabled           bool
	TemporalAddress           string
	TemporalNamespace         string
	TemporalTaskQueue         string
	TicketAPIBaseURL          string
	TicketAPIToken            string
	ApprovedToolFailOnApprove bool
	EvalRunFailAll            bool
	WorkerPollInterval        time.Duration
	WorkerShutdownTimeout     time.Duration
	LLMProvider               string
	LLMBaseURL                string
	LLMAPIKey                 string
	LLMModel                  string
	LLMTimeout                time.Duration
	EmbeddingProvider         string
	EmbeddingBaseURL          string
	EmbeddingAPIKey           string
	EmbeddingModel            string
	EmbeddingTimeout          time.Duration
}

// Load reads process configuration from environment variables and applies safe defaults.
func Load() (Config, error) {
	cfg := Config{
		Env:                       getEnv("OPSPILOT_ENV", defaultEnv),
		LogLevel:                  getEnv("OPSPILOT_LOG_LEVEL", defaultLogLevel),
		APIListenAddr:             getEnv("OPSPILOT_API_LISTEN_ADDR", defaultAPIListenAddr),
		PostgresDSN:               getEnv("OPSPILOT_POSTGRES_DSN", defaultPostgresDSN),
		EvalJudgeProvider:         getEnv("OPSPILOT_EVAL_JUDGE_PROVIDER", defaultEvalJudgeProvider),
		EvalJudgeBaseURL:          getEnv("OPSPILOT_EVAL_JUDGE_BASE_URL", ""),
		EvalJudgeAPIKey:           getEnv("OPSPILOT_EVAL_JUDGE_API_KEY", ""),
		EvalJudgeModel:            getEnv("OPSPILOT_EVAL_JUDGE_MODEL", ""),
		EvalJudgeTimeout:          defaultEvalJudgeTimeout,
		TemporalEnabled:           defaultTemporalEnabled,
		TemporalAddress:           getEnv("OPSPILOT_TEMPORAL_ADDRESS", defaultTemporalAddress),
		TemporalNamespace:         getEnv("OPSPILOT_TEMPORAL_NAMESPACE", defaultTemporalNamespace),
		TemporalTaskQueue:         getEnv("OPSPILOT_TEMPORAL_TASK_QUEUE", defaultTemporalTaskQueue),
		TicketAPIBaseURL:          getEnv("OPSPILOT_TICKET_API_BASE_URL", ""),
		TicketAPIToken:            getEnv("OPSPILOT_TICKET_API_TOKEN", ""),
		ApprovedToolFailOnApprove: defaultApprovedToolFailOnApprove,
		EvalRunFailAll:            defaultEvalRunFailAll,
		WorkerPollInterval:        defaultWorkerPollInterval,
		WorkerShutdownTimeout:     defaultWorkerShutdownTimeout,
		LLMProvider:               getEnv("OPSPILOT_LLM_PROVIDER", defaultLLMProvider),
		LLMBaseURL:                getEnv("OPSPILOT_LLM_BASE_URL", ""),
		LLMAPIKey:                 getEnv("OPSPILOT_LLM_API_KEY", ""),
		LLMModel:                  getEnv("OPSPILOT_LLM_MODEL", ""),
		LLMTimeout:                defaultLLMTimeout,
		EmbeddingProvider:         getEnv("OPSPILOT_EMBEDDING_PROVIDER", defaultEmbeddingProvider),
		EmbeddingBaseURL:          getEnv("OPSPILOT_EMBEDDING_BASE_URL", ""),
		EmbeddingAPIKey:           getEnv("OPSPILOT_EMBEDDING_API_KEY", ""),
		EmbeddingModel:            getEnv("OPSPILOT_EMBEDDING_MODEL", ""),
		EmbeddingTimeout:          defaultEmbeddingTimeout,
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
	if raw := os.Getenv("OPSPILOT_EVAL_RUN_FAIL_ALL"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_EVAL_RUN_FAIL_ALL: %w", err)
		}
		cfg.EvalRunFailAll = enabled
	}
	if raw := os.Getenv("OPSPILOT_EVAL_JUDGE_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_EVAL_JUDGE_TIMEOUT: %w", err)
		}
		cfg.EvalJudgeTimeout = timeout
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
	if cfg.EvalJudgeProvider == "" {
		return Config{}, fmt.Errorf("OPSPILOT_EVAL_JUDGE_PROVIDER must not be empty")
	}
	if cfg.EvalJudgeTimeout <= 0 {
		return Config{}, fmt.Errorf("OPSPILOT_EVAL_JUDGE_TIMEOUT must be positive")
	}
	if raw := os.Getenv("OPSPILOT_EMBEDDING_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_EMBEDDING_TIMEOUT: %w", err)
		}
		cfg.EmbeddingTimeout = timeout
	}
	if raw := os.Getenv("OPSPILOT_LLM_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse OPSPILOT_LLM_TIMEOUT: %w", err)
		}
		cfg.LLMTimeout = timeout
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

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opspilot-go/internal/app/config"
	"opspilot-go/internal/app/logging"
	storagepostgres "opspilot-go/internal/storage/postgres"
	"opspilot-go/internal/workflow"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := storagepostgres.OpenPool(context.Background(), cfg.PostgresDSN)
	if err != nil {
		logger.Error("open postgres pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	service := workflow.NewServiceWithStore(storagepostgres.NewWorkflowTaskStore(pool))
	runner := workflow.NewRunner(service, workflow.NewPlaceholderExecutor())

	logger.Info("worker booted",
		slog.String("env", cfg.Env),
		slog.Duration("poll_interval", cfg.WorkerPollInterval),
	)

	ticker := time.NewTicker(cfg.WorkerPollInterval)
	defer ticker.Stop()

	process := func() {
		processed, err := runner.ProcessNextBatch(ctx, 10)
		if err != nil {
			logger.Error("workflow batch failed", slog.Any("error", err))
			return
		}
		if processed > 0 {
			logger.Info("workflow batch processed", slog.Int("count", processed))
		}
	}

	process()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				process()
			}
		}
	}()

	<-ctx.Done()
	logger.Info("worker shutdown complete")
}

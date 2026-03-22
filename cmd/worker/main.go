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

	temporalworker "go.temporal.io/sdk/worker"
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
	executor := workflow.Executor(workflow.NewPlaceholderExecutor())

	var temporalWorker temporalworker.Worker
	if cfg.TemporalEnabled {
		temporalClient, err := workflow.DialTemporalClient(workflow.TemporalOptions{
			Address:   cfg.TemporalAddress,
			Namespace: cfg.TemporalNamespace,
			TaskQueue: cfg.TemporalTaskQueue,
		})
		if err != nil {
			logger.Error("dial temporal client", slog.Any("error", err))
			os.Exit(1)
		}
		defer temporalClient.Close()

		reportRunner := workflow.NewTemporalReportRunner(temporalClient, cfg.TemporalTaskQueue)
		approvedToolRunner := workflow.NewTemporalApprovedToolRunnerWithActivities(temporalClient, cfg.TemporalTaskQueue, &workflow.ApprovedToolActivities{
			FailOnApprove: cfg.ApprovedToolFailOnApprove,
		})
		temporalWorker = workflow.NewTemporalWorker(temporalClient, cfg.TemporalTaskQueue, reportRunner, approvedToolRunner)
		if err := temporalWorker.Start(); err != nil {
			logger.Error("start temporal worker", slog.Any("error", err))
			os.Exit(1)
		}
		defer temporalWorker.Stop()

		executor = workflow.NewTemporalExecutor(reportRunner, approvedToolRunner, executor)
		logger.Info("temporal worker booted",
			slog.String("address", cfg.TemporalAddress),
			slog.String("namespace", cfg.TemporalNamespace),
			slog.String("task_queue", cfg.TemporalTaskQueue),
			slog.Bool("approved_tool_fail_on_approve", cfg.ApprovedToolFailOnApprove),
		)
	}

	runner := workflow.NewRunner(service, executor)

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

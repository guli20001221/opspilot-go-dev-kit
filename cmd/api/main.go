package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opspilot-go/internal/app/config"
	"opspilot-go/internal/app/httpapi"
	"opspilot-go/internal/app/logging"
	"opspilot-go/internal/report"
	storagepostgres "opspilot-go/internal/storage/postgres"
	toolregistry "opspilot-go/internal/tools/registry"
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

	pool, err := storagepostgres.OpenPool(context.Background(), cfg.PostgresDSN)
	if err != nil {
		logger.Error("open postgres pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	var taskStarter workflow.TaskStarter
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

		taskStarter = workflow.NewTemporalApprovedToolRunner(temporalClient, cfg.TemporalTaskQueue)
		logger.Info("api temporal client booted",
			slog.String("address", cfg.TemporalAddress),
			slog.String("namespace", cfg.TemporalNamespace),
			slog.String("task_queue", cfg.TemporalTaskQueue),
		)
	}

	workflowService := workflow.NewServiceWithHooks(storagepostgres.NewWorkflowTaskStore(pool), taskStarter)
	reportService := report.NewServiceWithStore(storagepostgres.NewReportStore(pool))
	registry := toolregistry.NewDefaultRegistryWithOptions(toolregistry.Options{
		TicketAPIBaseURL: cfg.TicketAPIBaseURL,
		TicketAPIToken:   cfg.TicketAPIToken,
	})

	server := &http.Server{
		Addr:              cfg.APIListenAddr,
		Handler:           httpapi.NewHandlerWithDependencies(httpapi.Dependencies{Workflows: workflowService, Reports: reportService, Registry: registry}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("api listening",
			slog.String("addr", cfg.APIListenAddr),
			slog.String("env", cfg.Env),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.WorkerShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("api shutdown complete")
}

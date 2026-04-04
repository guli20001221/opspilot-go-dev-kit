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
	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
	"opspilot-go/internal/observability/tracedetail"
	"opspilot-go/internal/report"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
	storagepostgres "opspilot-go/internal/storage/postgres"
	toolregistry "opspilot-go/internal/tools/registry"
	"opspilot-go/internal/version"
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

	sessionService := session.NewServiceWithStore(storagepostgres.NewSessionStore(pool))
	retrievalStore := storagepostgres.NewRetrievalChunkStore(pool, &retrieval.PlaceholderEmbedder{})
	versionService := version.NewServiceWithStore(storagepostgres.NewVersionStore(pool))
	workflowService := workflow.NewServiceWithDependencies(storagepostgres.NewWorkflowTaskStore(pool), taskStarter, versionService)
	reportService := report.NewServiceWithDependencies(storagepostgres.NewReportStore(pool), versionService)
	caseService := casesvc.NewServiceWithStore(storagepostgres.NewCaseStore(pool))
	traceDetails := tracedetail.NewService(workflowService, reportService, caseService)
	evalCaseService := evalsvc.NewServiceWithStore(storagepostgres.NewEvalCaseStore(pool), caseService, traceDetails)
	evalDatasetService := evalsvc.NewDatasetServiceWithStore(storagepostgres.NewEvalDatasetStore(pool), evalCaseService)
	evalRunService := evalsvc.NewRunServiceWithStore(storagepostgres.NewEvalRunStore(pool), evalDatasetService)
	evalReportService := evalsvc.NewEvalReportServiceWithDependencies(storagepostgres.NewEvalReportStore(pool), evalRunService)
	registry := toolregistry.NewDefaultRegistryWithOptions(toolregistry.Options{
		TicketAPIBaseURL: cfg.TicketAPIBaseURL,
		TicketAPIToken:   cfg.TicketAPIToken,
	})

	server := &http.Server{
		Addr:              cfg.APIListenAddr,
		Handler:           httpapi.NewHandlerWithDependencies(httpapi.Dependencies{Workflows: workflowService, Reports: reportService, Cases: caseService, EvalCases: evalCaseService, EvalDatasets: evalDatasetService, EvalRuns: evalRunService, EvalReports: evalReportService, Versions: versionService, Sessions: sessionService, Retrieval: retrievalStore, Registry: registry}),
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

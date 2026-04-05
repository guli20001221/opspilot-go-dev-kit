package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	agenttool "opspilot-go/internal/agent/tool"
	appchat "opspilot-go/internal/app/chat"
	"opspilot-go/internal/app/config"
	"opspilot-go/internal/app/logging"
	"opspilot-go/internal/observability/metrics"
	"opspilot-go/internal/observability/tracing"
	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/eval"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/report"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
	storagepostgres "opspilot-go/internal/storage/postgres"
	toolregistry "opspilot-go/internal/tools/registry"
	"opspilot-go/internal/version"
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

	shutdownTracer := tracing.InitStdout()
	defer shutdownTracer(context.Background())
	shutdownMetrics := metrics.InitStdout()
	defer shutdownMetrics(context.Background())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := storagepostgres.OpenPool(context.Background(), cfg.PostgresDSN)
	if err != nil {
		logger.Error("open postgres pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	sessionService := session.NewServiceWithStore(storagepostgres.NewSessionStore(pool))
	contextEngine := contextengine.NewService(contextengine.Config{})
	embedder, err := retrieval.NewConfiguredEmbedder(retrieval.EmbedderOptions{
		Provider: cfg.EmbeddingProvider,
		BaseURL:  cfg.EmbeddingBaseURL,
		APIKey:   cfg.EmbeddingAPIKey,
		Model:    cfg.EmbeddingModel,
		Timeout:  cfg.EmbeddingTimeout,
	})
	if err != nil {
		logger.Error("configure embedder", slog.Any("error", err))
		os.Exit(1)
	}
	retrievalStore := storagepostgres.NewRetrievalChunkStore(pool, embedder)
	versionService := version.NewServiceWithStore(storagepostgres.NewVersionStore(pool))
	service := workflow.NewServiceWithDependencies(storagepostgres.NewWorkflowTaskStore(pool), nil, versionService)
	reportService := report.NewServiceWithDependencies(storagepostgres.NewReportStore(pool), versionService)
	executor := workflow.Executor(workflow.NewPlaceholderExecutor())
	llmProvider, err := llm.NewConfiguredProvider(llm.ProviderOptions{
		Provider: cfg.LLMProvider,
		BaseURL:  cfg.LLMBaseURL,
		APIKey:   cfg.LLMAPIKey,
		Model:    cfg.LLMModel,
		Timeout:  cfg.LLMTimeout,
	})
	if err != nil {
		logger.Error("configure llm provider", slog.Any("error", err))
		os.Exit(1)
	}
	evalJudge, err := eval.NewConfiguredJudge(eval.JudgeOptions{
		Provider:   cfg.EvalJudgeProvider,
		BaseURL:    cfg.EvalJudgeBaseURL,
		APIKey:     cfg.EvalJudgeAPIKey,
		Model:      cfg.EvalJudgeModel,
		PromptPath: eval.PlaceholderJudgePromptPath,
		Timeout:    cfg.EvalJudgeTimeout,
	})
	if err != nil {
		logger.Error("configure eval judge", slog.Any("error", err))
		os.Exit(1)
	}
	evalRunService := eval.NewRunServiceWithDependencies(storagepostgres.NewEvalRunStore(pool), nil, evalJudge)
	evalReportService := eval.NewEvalReportServiceWithDependencies(storagepostgres.NewEvalReportStore(pool), evalRunService)
	registry := toolregistry.NewDefaultRegistryWithOptions(toolregistry.Options{
		TicketAPIBaseURL: cfg.TicketAPIBaseURL,
		TicketAPIToken:   cfg.TicketAPIToken,
	})
	tools := agenttool.NewService(registry)
	// Eval chat service uses a separate workflow service (nil) to prevent
	// eval runs from promoting async tasks as a side effect.
	evalChatService := appchat.NewServiceWithLLM(sessionService, nil, registry, retrievalStore, llmProvider)

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

		reportActivities := workflow.NewReportActivities(sessionService, contextEngine, retrievalStore)
		reportRunner := workflow.NewTemporalReportRunnerWithActivities(temporalClient, cfg.TemporalTaskQueue, reportActivities)
		activities := workflow.NewApprovedToolActivities(tools)
		activities.FailOnApprove = cfg.ApprovedToolFailOnApprove
		approvedToolRunner := workflow.NewTemporalApprovedToolRunnerWithActivities(temporalClient, cfg.TemporalTaskQueue, activities)
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
	var evalRunExecutor eval.RunExecutor
	if cfg.EvalRunFailAll {
		placeholder := eval.NewPlaceholderRunExecutor()
		placeholder.FailAll = true
		evalRunExecutor = placeholder
	} else {
		evalRunExecutor = eval.NewChatRunExecutor(evalChatService, evalRunService)
	}

	runner := workflow.NewRunnerWithReports(service, executor, reportService)
	evalRunner := eval.NewRunnerWithReports(evalRunService, evalRunExecutor, evalReportService)

	logger.Info("worker booted",
		slog.String("env", cfg.Env),
		slog.String("eval_judge_provider", cfg.EvalJudgeProvider),
		slog.String("eval_judge_model", cfg.EvalJudgeModel),
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
		evalProcessed, err := evalRunner.ProcessNextBatch(ctx, 10)
		if err != nil {
			logger.Error("eval run batch failed", slog.Any("error", err))
			return
		}
		if evalProcessed > 0 {
			logger.Info("eval run batch processed", slog.Int("count", evalProcessed))
		}
	}

	process()

	done := startPollLoop(ctx, ticker.C, process)

	<-ctx.Done()
	<-done
	logger.Info("worker shutdown complete")
}

func startPollLoop(ctx context.Context, ticks <-chan time.Time, process func()) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticks:
				process()
			}
		}
	}()

	return done
}

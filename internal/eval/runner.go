package eval

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const finalizationTimeout = 5 * time.Second

// RunExecutor performs the execution body for one claimed eval run.
type RunExecutor interface {
	ExecuteRun(ctx context.Context, run EvalRun) error
}

// Runner claims queued eval runs and advances them to terminal states.
type Runner struct {
	service  *RunService
	executor RunExecutor
	reports  *EvalReportService
}

// NewRunner constructs an eval-run worker runner.
func NewRunner(service *RunService, executor RunExecutor) *Runner {
	return NewRunnerWithReports(service, executor, nil)
}

// NewRunnerWithReports constructs an eval-run worker runner with optional eval-report materialization.
func NewRunnerWithReports(service *RunService, executor RunExecutor, reports *EvalReportService) *Runner {
	if service == nil {
		service = NewRunServiceWithStore(nil, nil)
	}
	if executor == nil {
		executor = NewPlaceholderRunExecutor()
	}

	return &Runner{
		service:  service,
		executor: executor,
		reports:  reports,
	}
}

// ProcessNextBatch claims and executes up to limit queued eval runs.
func (r *Runner) ProcessNextBatch(ctx context.Context, limit int) (int, error) {
	runs, err := r.service.ClaimQueuedRuns(ctx, limit)
	if err != nil {
		return 0, err
	}

	for _, run := range runs {
		if err := r.executor.ExecuteRun(ctx, run); err != nil {
			finalizeCtx, cancel := finalizationContext(ctx)
			reason := summarizeRunExecutionError(err)
			_, markErr := r.service.MarkRunFailed(finalizeCtx, run.ID, reason)
			if markErr != nil {
				_, markErr = r.service.MarkRunFailedWithFallback(finalizeCtx, run.ID, summarizeRunJudgeFailure(reason, markErr))
			}
			cancel()
			if markErr != nil {
				return 0, markErr
			}
			if err := r.materializeRunReport(finalizeCtx, run.ID); err != nil {
				cancel()
				return 0, err
			}
			cancel()
			continue
		}
		finalizeCtx, cancel := finalizationContext(ctx)
		_, err := r.service.MarkRunSucceeded(finalizeCtx, run.ID)
		if err != nil {
			_, fallbackErr := r.service.MarkRunFailedWithFallback(finalizeCtx, run.ID, summarizeRunJudgeFailure("eval judge failed after successful execution", err))
			cancel()
			if fallbackErr != nil {
				return 0, fallbackErr
			}
			if err := r.materializeRunReport(finalizeCtx, run.ID); err != nil {
				cancel()
				return 0, err
			}
			cancel()
			continue
		}
		if err := r.materializeRunReport(finalizeCtx, run.ID); err != nil {
			cancel()
			return 0, err
		}
		cancel()
	}

	return len(runs), nil
}

// PlaceholderRunExecutor advances eval runs without judge or model execution.
type PlaceholderRunExecutor struct {
	FailAll bool
}

// NewPlaceholderRunExecutor constructs the eval placeholder executor.
func NewPlaceholderRunExecutor() *PlaceholderRunExecutor {
	return &PlaceholderRunExecutor{}
}

// ExecuteRun performs placeholder eval-run execution.
func (e *PlaceholderRunExecutor) ExecuteRun(_ context.Context, run EvalRun) error {
	if e.FailAll {
		return fmt.Errorf("fault injection: eval run failed for %s", run.ID)
	}
	return nil
}

func summarizeRunExecutionError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func summarizeRunJudgeFailure(prefix string, err error) string {
	summary := strings.TrimSpace(prefix)
	if err == nil {
		return summary
	}
	if summary == "" {
		return err.Error()
	}
	return fmt.Sprintf("%s: %v", summary, err)
}

func (r *Runner) materializeRunReport(ctx context.Context, runID string) error {
	if r.reports == nil {
		return nil
	}
	if _, err := r.reports.MaterializeRunReport(ctx, runID); err != nil {
		return fmt.Errorf("materialize eval report: %w", err)
	}
	return nil
}

func finalizationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), finalizationTimeout)
}

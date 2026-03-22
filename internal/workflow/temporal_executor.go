package workflow

import (
	"context"
)

// ReportWorkflowRunner starts and waits for Temporal-backed report workflows.
type ReportWorkflowRunner interface {
	RunReportWorkflow(ctx context.Context, task Task) (ExecutionResult, error)
}

// TemporalExecutor routes report-generation tasks through Temporal and falls
// back to the existing executor for other task types.
type TemporalExecutor struct {
	reportRunner ReportWorkflowRunner
	fallback     Executor
}

// NewTemporalExecutor constructs the Temporal-aware task executor.
func NewTemporalExecutor(reportRunner ReportWorkflowRunner, fallback Executor) *TemporalExecutor {
	if fallback == nil {
		fallback = NewPlaceholderExecutor()
	}

	return &TemporalExecutor{
		reportRunner: reportRunner,
		fallback:     fallback,
	}
}

// Execute routes supported tasks to Temporal-backed workflows when configured.
func (e *TemporalExecutor) Execute(ctx context.Context, task Task) (ExecutionResult, error) {
	if task.TaskType == TaskTypeReportGeneration && e.reportRunner != nil {
		return e.reportRunner.RunReportWorkflow(ctx, task)
	}

	return e.fallback.Execute(ctx, task)
}

package workflow

import (
	"context"
	"strings"
)

// ReportWorkflowRunner starts and waits for Temporal-backed report workflows.
type ReportWorkflowRunner interface {
	RunReportWorkflow(ctx context.Context, task Task) (ExecutionResult, error)
}

// ApprovedToolWorkflowRunner continues the Temporal-backed approval workflow
// after the API has recorded an approval or retry action.
type ApprovedToolWorkflowRunner interface {
	ContinueApprovedToolWorkflow(ctx context.Context, task Task) (ExecutionResult, error)
}

// TemporalExecutor routes report-generation tasks through Temporal and falls
// back to the existing executor for other task types.
type TemporalExecutor struct {
	reportRunner   ReportWorkflowRunner
	approvalRunner ApprovedToolWorkflowRunner
	fallback       Executor
}

// NewTemporalExecutor constructs the Temporal-aware task executor.
func NewTemporalExecutor(reportRunner ReportWorkflowRunner, approvalRunner ApprovedToolWorkflowRunner, fallback Executor) *TemporalExecutor {
	if fallback == nil {
		fallback = NewPlaceholderExecutor()
	}

	return &TemporalExecutor{
		reportRunner:   reportRunner,
		approvalRunner: approvalRunner,
		fallback:       fallback,
	}
}

// Execute routes supported tasks to Temporal-backed workflows when configured.
func (e *TemporalExecutor) Execute(ctx context.Context, task Task) (ExecutionResult, error) {
	if task.TaskType == TaskTypeReportGeneration && e.reportRunner != nil {
		return e.reportRunner.RunReportWorkflow(ctx, task)
	}
	if task.TaskType == TaskTypeApprovedToolExecution &&
		e.approvalRunner != nil &&
		(strings.HasPrefix(task.AuditRef, "approval:") || strings.HasPrefix(task.AuditRef, "retry:")) {
		return e.approvalRunner.ContinueApprovedToolWorkflow(ctx, task)
	}

	return e.fallback.Execute(ctx, task)
}

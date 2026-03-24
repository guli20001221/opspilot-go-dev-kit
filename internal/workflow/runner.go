package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ExecutionResult captures the worker-visible outcome metadata.
type ExecutionResult struct {
	AuditRef string
	Detail   string
}

// Executor performs the side-effecting task body behind the workflow orchestration.
type Executor interface {
	Execute(ctx context.Context, task Task) (ExecutionResult, error)
}

// ReportRecorder persists durable reports emitted by successful workflow tasks.
type ReportRecorder interface {
	RecordGeneratedReport(ctx context.Context, task Task, result ExecutionResult) (string, error)
}

// AtomicReportRecorder persists report rows and the task success transition in
// one storage operation when the underlying store supports it.
type AtomicReportRecorder interface {
	ReportRecorder
	SupportsAtomicFinalization() bool
	FinalizeGeneratedReportTask(ctx context.Context, task Task, result ExecutionResult, event AuditEvent) (Task, string, error)
}

// Runner claims queued tasks and advances them through the placeholder workflow path.
type Runner struct {
	service  *Service
	executor Executor
	reports  ReportRecorder
}

// NewRunner constructs a workflow runner.
func NewRunner(service *Service, executor Executor) *Runner {
	return NewRunnerWithReports(service, executor, nil)
}

// NewRunnerWithReports constructs a workflow runner with optional report persistence.
func NewRunnerWithReports(service *Service, executor Executor, reports ReportRecorder) *Runner {
	if service == nil {
		service = NewService()
	}
	if executor == nil {
		executor = NewPlaceholderExecutor()
	}

	return &Runner{
		service:  service,
		executor: executor,
		reports:  reports,
	}
}

// ProcessNextBatch claims and executes up to limit queued tasks.
func (r *Runner) ProcessNextBatch(ctx context.Context, limit int) (int, error) {
	tasks, err := r.service.ClaimQueuedTasks(ctx, limit)
	if err != nil {
		return 0, err
	}

	for _, task := range tasks {
		result, execErr := r.executor.Execute(ctx, task)

		if execErr != nil {
			task.Status = StatusFailed
			task.ErrorReason = summarizeExecutionError(execErr)
			task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_failed")
		} else {
			task.Status = StatusSucceeded
			task.ErrorReason = ""
			task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_succeeded")
		}

		task.UpdatedAt = time.Now().UTC()
		action := AuditActionSucceeded
		if execErr == nil && task.TaskType == TaskTypeReportGeneration && r.reports != nil {
			if task.VersionID == "" && r.service.versionSource != nil {
				versionID, err := r.service.versionSource.CurrentVersionID(ctx)
				if err != nil {
					task.Status = StatusFailed
					task.ErrorReason = summarizeExecutionError(err)
					task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_failed")
					task.UpdatedAt = time.Now().UTC()
					action = AuditActionFailed
				} else {
					task.VersionID = versionID
				}
			}
			successEvent := AuditEvent{
				TaskID:    task.ID,
				Action:    AuditActionSucceeded,
				Actor:     "worker",
				Detail:    successOrFailureDetail(AuditActionSucceeded, result.Detail, "", task.Status),
				CreatedAt: task.UpdatedAt,
			}
			if finalizer, ok := r.reports.(AtomicReportRecorder); ok && finalizer.SupportsAtomicFinalization() {
				if _, _, err := finalizer.FinalizeGeneratedReportTask(ctx, task, result, successEvent); err == nil {
					continue
				} else {
					task.Status = StatusFailed
					task.ErrorReason = summarizeExecutionError(err)
					task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_failed")
					task.UpdatedAt = time.Now().UTC()
					action = AuditActionFailed
				}
			} else {
				if _, err := r.reports.RecordGeneratedReport(ctx, task, result); err != nil {
					task.Status = StatusFailed
					task.ErrorReason = summarizeExecutionError(err)
					task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_failed")
					task.UpdatedAt = time.Now().UTC()
					action = AuditActionFailed
				}
			}
		} else if execErr != nil {
			action = AuditActionFailed
		}

		if _, err := r.service.store.UpdateTaskWithEvent(ctx, task, AuditEvent{
			TaskID:    task.ID,
			Action:    action,
			Actor:     "worker",
			Detail:    successOrFailureDetail(action, result.Detail, task.ErrorReason, task.Status),
			CreatedAt: task.UpdatedAt,
		}); err != nil {
			return 0, err
		}
	}

	return len(tasks), nil
}

func fallbackAuditRef(value string, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

func summarizeExecutionError(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}

	if idx := strings.LastIndex(message, "): "); idx >= 0 && idx+3 < len(message) {
		message = message[idx+3:]
	}
	if idx := strings.Index(message, " (type: "); idx >= 0 {
		message = message[:idx]
	}

	return strings.TrimSpace(message)
}

func successOrFailureDetail(action string, successDetail string, failureDetail string, fallback string) string {
	if action == AuditActionSucceeded {
		return fallbackAuditRef(successDetail, fallback)
	}

	return fallbackAuditRef(classifyExecutionFailure(failureDetail), fallback)
}

func classifyExecutionFailure(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}

	lower := strings.ToLower(summary)
	switch {
	case strings.Contains(lower, "fault injection"):
		return "injected_failure: " + summary
	case strings.Contains(lower, "requires ticket_"), strings.Contains(lower, "requires comment"), strings.Contains(lower, "requires query"), strings.Contains(lower, "decode ticket_"):
		return "validation_error: " + summary
	case strings.Contains(lower, "status 401"), strings.Contains(lower, "status 403"), strings.Contains(lower, "unauthorized"):
		return "authorization_error: " + summary
	case strings.Contains(lower, "status 4"):
		return "request_error: " + summary
	case strings.Contains(lower, "status 5"), strings.Contains(lower, "call ticket_"):
		return "upstream_error: " + summary
	default:
		return "execution_error: " + summary
	}
}

// PlaceholderExecutor advances known task types without external side effects.
type PlaceholderExecutor struct{}

// NewPlaceholderExecutor constructs the worker placeholder executor.
func NewPlaceholderExecutor() *PlaceholderExecutor {
	return &PlaceholderExecutor{}
}

// Execute performs placeholder task execution for known task types.
func (e *PlaceholderExecutor) Execute(_ context.Context, task Task) (ExecutionResult, error) {
	switch task.TaskType {
	case TaskTypeReportGeneration:
		return ExecutionResult{
			AuditRef: "worker:placeholder_report_generation",
			Detail:   "report_generation completed",
		}, nil
	case TaskTypeApprovedToolExecution:
		return ExecutionResult{
			AuditRef: "worker:placeholder_approved_tool_execution",
			Detail:   "approved_tool_execution completed",
		}, nil
	default:
		return ExecutionResult{AuditRef: "worker:placeholder_failed"}, fmt.Errorf("unsupported task type: %s", task.TaskType)
	}
}

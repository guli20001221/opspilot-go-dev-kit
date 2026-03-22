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
}

// Executor performs the side-effecting task body behind the workflow orchestration.
type Executor interface {
	Execute(ctx context.Context, task Task) (ExecutionResult, error)
}

// Runner claims queued tasks and advances them through the placeholder workflow path.
type Runner struct {
	service  *Service
	executor Executor
}

// NewRunner constructs a workflow runner.
func NewRunner(service *Service, executor Executor) *Runner {
	if service == nil {
		service = NewService()
	}
	if executor == nil {
		executor = NewPlaceholderExecutor()
	}

	return &Runner{
		service:  service,
		executor: executor,
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
		action := AuditActionSucceeded
		if execErr != nil {
			task.Status = StatusFailed
			task.ErrorReason = summarizeExecutionError(execErr)
			task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_failed")
			action = AuditActionFailed
		} else {
			task.Status = StatusSucceeded
			task.ErrorReason = ""
			task.AuditRef = fallbackAuditRef(result.AuditRef, "worker:placeholder_succeeded")
		}

		task.UpdatedAt = time.Now().UTC()
		if _, err := r.service.store.UpdateTaskWithEvent(ctx, task, AuditEvent{
			TaskID:    task.ID,
			Action:    action,
			Actor:     "worker",
			Detail:    fallbackAuditRef(task.ErrorReason, task.Status),
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
		return ExecutionResult{AuditRef: "worker:placeholder_report_generation"}, nil
	case TaskTypeApprovedToolExecution:
		return ExecutionResult{AuditRef: "worker:placeholder_approved_tool_execution"}, nil
	default:
		return ExecutionResult{AuditRef: "worker:placeholder_failed"}, fmt.Errorf("unsupported task type: %s", task.TaskType)
	}
}

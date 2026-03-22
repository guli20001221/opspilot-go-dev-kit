package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/workflow"
)

// WorkflowTaskStore persists workflow task records in PostgreSQL.
type WorkflowTaskStore struct {
	pool *pgxpool.Pool
}

// NewWorkflowTaskStore constructs the workflow task repository.
func NewWorkflowTaskStore(pool *pgxpool.Pool) *WorkflowTaskStore {
	return &WorkflowTaskStore{pool: pool}
}

// SaveTask inserts a newly promoted task record.
func (s *WorkflowTaskStore) SaveTask(ctx context.Context, task workflow.Task) (workflow.Task, error) {
	const query = `
INSERT INTO workflow_tasks (
    id,
    request_id,
    tenant_id,
    session_id,
    task_type,
    status,
    reason,
    error_reason,
    audit_ref,
    requires_approval,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)`

	_, err := s.pool.Exec(ctx, query,
		task.ID,
		task.RequestID,
		task.TenantID,
		task.SessionID,
		task.TaskType,
		task.Status,
		task.Reason,
		task.ErrorReason,
		task.AuditRef,
		task.RequiresApproval,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		return workflow.Task{}, fmt.Errorf("insert workflow task: %w", err)
	}

	return task, nil
}

// GetTask loads a task by ID.
func (s *WorkflowTaskStore) GetTask(ctx context.Context, taskID string) (workflow.Task, error) {
	const query = `
SELECT
    id,
    request_id,
    tenant_id,
    session_id,
    task_type,
    status,
    reason,
    error_reason,
    audit_ref,
    requires_approval,
    created_at,
    updated_at
FROM workflow_tasks
WHERE id = $1`

	var task workflow.Task
	err := s.pool.QueryRow(ctx, query, taskID).Scan(
		&task.ID,
		&task.RequestID,
		&task.TenantID,
		&task.SessionID,
		&task.TaskType,
		&task.Status,
		&task.Reason,
		&task.ErrorReason,
		&task.AuditRef,
		&task.RequiresApproval,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return workflow.Task{}, fmt.Errorf("%w: %s", workflow.ErrTaskNotFound, taskID)
		}

		return workflow.Task{}, fmt.Errorf("select workflow task: %w", err)
	}

	return task, nil
}

// ClaimQueuedTasks atomically marks queued tasks as running and returns them.
func (s *WorkflowTaskStore) ClaimQueuedTasks(ctx context.Context, limit int) ([]workflow.Task, error) {
	const query = `
WITH next_tasks AS (
    SELECT id
    FROM workflow_tasks
    WHERE status = $1
    ORDER BY created_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE workflow_tasks AS t
SET status = $3, updated_at = $4
FROM next_tasks
WHERE t.id = next_tasks.id
RETURNING
    t.id,
    t.request_id,
    t.tenant_id,
    t.session_id,
    t.task_type,
    t.status,
    t.reason,
    t.error_reason,
    t.audit_ref,
    t.requires_approval,
    t.created_at,
    t.updated_at`

	now := time.Now().UTC()
	rows, err := s.pool.Query(ctx, query, workflow.StatusQueued, limit, workflow.StatusRunning, now)
	if err != nil {
		return nil, fmt.Errorf("claim workflow tasks: %w", err)
	}
	defer rows.Close()

	var tasks []workflow.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed workflow tasks: %w", err)
	}

	return tasks, nil
}

// UpdateTask persists task state after worker processing.
func (s *WorkflowTaskStore) UpdateTask(ctx context.Context, task workflow.Task) (workflow.Task, error) {
	const query = `
UPDATE workflow_tasks
SET
    status = $2,
    error_reason = $3,
    audit_ref = $4,
    updated_at = $5
WHERE id = $1
RETURNING
    id,
    request_id,
    tenant_id,
    session_id,
    task_type,
    status,
    reason,
    error_reason,
    audit_ref,
    requires_approval,
    created_at,
    updated_at`

	row := s.pool.QueryRow(ctx, query, task.ID, task.Status, task.ErrorReason, task.AuditRef, task.UpdatedAt)
	updated, err := scanTask(row)
	if err != nil {
		return workflow.Task{}, err
	}

	return updated, nil
}

// AppendTaskEvent inserts a structured task audit record.
func (s *WorkflowTaskStore) AppendTaskEvent(ctx context.Context, event workflow.AuditEvent) (workflow.AuditEvent, error) {
	const query = `
INSERT INTO workflow_task_events (
    task_id,
    action,
    actor,
    detail,
    created_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id`

	err := s.pool.QueryRow(ctx, query, event.TaskID, event.Action, event.Actor, event.Detail, event.CreatedAt).Scan(&event.ID)
	if err != nil {
		return workflow.AuditEvent{}, fmt.Errorf("insert workflow task event: %w", err)
	}

	return event, nil
}

// ListTaskEvents returns the audit history for a task.
func (s *WorkflowTaskStore) ListTaskEvents(ctx context.Context, taskID string) ([]workflow.AuditEvent, error) {
	const query = `
SELECT
    id,
    task_id,
    action,
    actor,
    detail,
    created_at
FROM workflow_task_events
WHERE task_id = $1
ORDER BY created_at, id`

	rows, err := s.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("select workflow task events: %w", err)
	}
	defer rows.Close()

	var events []workflow.AuditEvent
	for rows.Next() {
		var event workflow.AuditEvent
		if err := rows.Scan(&event.ID, &event.TaskID, &event.Action, &event.Actor, &event.Detail, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow task event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow task events: %w", err)
	}

	return events, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(row taskScanner) (workflow.Task, error) {
	var task workflow.Task
	err := row.Scan(
		&task.ID,
		&task.RequestID,
		&task.TenantID,
		&task.SessionID,
		&task.TaskType,
		&task.Status,
		&task.Reason,
		&task.ErrorReason,
		&task.AuditRef,
		&task.RequiresApproval,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return workflow.Task{}, fmt.Errorf("%w", workflow.ErrTaskNotFound)
		}

		return workflow.Task{}, fmt.Errorf("scan workflow task: %w", err)
	}

	return task, nil
}

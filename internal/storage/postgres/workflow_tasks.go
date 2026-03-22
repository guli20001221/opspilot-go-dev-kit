package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/workflow"
)

// WorkflowTaskStore persists workflow task records in PostgreSQL.
type WorkflowTaskStore struct {
	pool *pgxpool.Pool
}

type taskQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewWorkflowTaskStore constructs the workflow task repository.
func NewWorkflowTaskStore(pool *pgxpool.Pool) *WorkflowTaskStore {
	return &WorkflowTaskStore{pool: pool}
}

// SaveTask inserts a newly promoted task record.
func (s *WorkflowTaskStore) SaveTask(ctx context.Context, task workflow.Task) (workflow.Task, error) {
	if err := s.insertTask(ctx, s.pool, task); err != nil {
		return workflow.Task{}, fmt.Errorf("insert workflow task: %w", err)
	}

	return task, nil
}

// CreateTaskWithEvent inserts a task and its initial audit record atomically.
func (s *WorkflowTaskStore) CreateTaskWithEvent(ctx context.Context, task workflow.Task, event workflow.AuditEvent) (workflow.Task, error) {
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		if err := s.insertTask(ctx, tx, task); err != nil {
			return fmt.Errorf("insert workflow task: %w", err)
		}
		if _, err := s.appendTaskEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return workflow.Task{}, err
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
	if limit <= 0 {
		return nil, nil
	}

	var tasks []workflow.Task
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		tasks, err = s.claimQueuedTasks(ctx, tx, limit)
		return err
	}); err != nil {
		return nil, err
	}

	return tasks, nil
}

// UpdateTask persists task state after worker processing.
func (s *WorkflowTaskStore) UpdateTask(ctx context.Context, task workflow.Task) (workflow.Task, error) {
	row := s.pool.QueryRow(ctx, updateTaskQuery, task.ID, task.Status, task.ErrorReason, task.AuditRef, task.UpdatedAt)
	updated, err := scanTask(row)
	if err != nil {
		return workflow.Task{}, err
	}

	return updated, nil
}

// UpdateTaskWithEvent updates a task and appends an audit record atomically.
func (s *WorkflowTaskStore) UpdateTaskWithEvent(ctx context.Context, task workflow.Task, event workflow.AuditEvent) (workflow.Task, error) {
	var updated workflow.Task
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, updateTaskQuery, task.ID, task.Status, task.ErrorReason, task.AuditRef, task.UpdatedAt)
		var err error
		updated, err = scanTask(row)
		if err != nil {
			return err
		}
		if _, err := s.appendTaskEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return workflow.Task{}, err
	}

	return updated, nil
}

// AppendTaskEvent inserts a structured task audit record.
func (s *WorkflowTaskStore) AppendTaskEvent(ctx context.Context, event workflow.AuditEvent) (workflow.AuditEvent, error) {
	return s.appendTaskEvent(ctx, s.pool, event)
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

const claimQueuedTasksQuery = `
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

const updateTaskQuery = `
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

const insertTaskEventQuery = `
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

const insertTaskQuery = `
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

func (s *WorkflowTaskStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workflow task transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit workflow task transaction: %w", err)
	}

	return nil
}

func (s *WorkflowTaskStore) insertTask(ctx context.Context, db taskQuerier, task workflow.Task) error {
	_, err := db.Exec(ctx, insertTaskQuery,
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
	return err
}

func (s *WorkflowTaskStore) appendTaskEvent(ctx context.Context, db taskQuerier, event workflow.AuditEvent) (workflow.AuditEvent, error) {
	err := db.QueryRow(ctx, insertTaskEventQuery, event.TaskID, event.Action, event.Actor, event.Detail, event.CreatedAt).Scan(&event.ID)
	if err != nil {
		return workflow.AuditEvent{}, fmt.Errorf("insert workflow task event: %w", err)
	}

	return event, nil
}

func (s *WorkflowTaskStore) claimQueuedTasks(ctx context.Context, tx pgx.Tx, limit int) ([]workflow.Task, error) {
	now := time.Now().UTC()
	rows, err := tx.Query(ctx, claimQueuedTasksQuery, workflow.StatusQueued, limit, workflow.StatusRunning, now)
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

	for _, task := range tasks {
		if _, err := s.appendTaskEvent(ctx, tx, workflow.AuditEvent{
			TaskID:    task.ID,
			Action:    workflow.AuditActionClaimed,
			Actor:     "worker",
			Detail:    task.Status,
			CreatedAt: task.UpdatedAt,
		}); err != nil {
			return nil, err
		}
	}

	return tasks, nil
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

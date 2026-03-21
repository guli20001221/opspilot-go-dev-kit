package postgres

import (
	"context"
	"errors"
	"fmt"

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

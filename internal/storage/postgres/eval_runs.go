package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

// EvalRunStore persists durable eval runs in PostgreSQL.
type EvalRunStore struct {
	pool *pgxpool.Pool
}

// NewEvalRunStore constructs the eval run repository.
func NewEvalRunStore(pool *pgxpool.Pool) *EvalRunStore {
	return &EvalRunStore{pool: pool}
}

// CreateRun inserts one durable eval run row.
func (s *EvalRunStore) CreateRun(ctx context.Context, item evalsvc.EvalRun) (evalsvc.EvalRun, error) {
	const query = `
INSERT INTO eval_runs (
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULL, NULL
)`
	if _, err := s.pool.Exec(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.DatasetID,
		item.DatasetName,
		item.DatasetItemCount,
		item.Status,
		item.CreatedBy,
		item.ErrorReason,
		item.CreatedAt,
		item.UpdatedAt,
	); err != nil {
		return evalsvc.EvalRun{}, fmt.Errorf("insert eval run: %w", err)
	}
	return s.GetRun(ctx, item.ID)
}

// GetRun loads one durable eval run by ID.
func (s *EvalRunStore) GetRun(ctx context.Context, runID string) (evalsvc.EvalRun, error) {
	const query = `
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM eval_runs
WHERE id = $1`

	var item evalsvc.EvalRun
	var startedAt, finishedAt sql.NullTime
	if err := s.pool.QueryRow(ctx, query, runID).Scan(
		&item.ID,
		&item.TenantID,
		&item.DatasetID,
		&item.DatasetName,
		&item.DatasetItemCount,
		&item.Status,
		&item.CreatedBy,
		&item.ErrorReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalRun{}, evalsvc.ErrEvalRunNotFound
		}
		return evalsvc.EvalRun{}, fmt.Errorf("scan eval run: %w", err)
	}
	if startedAt.Valid {
		item.StartedAt = startedAt.Time
	}
	if finishedAt.Valid {
		item.FinishedAt = finishedAt.Time
	}
	return item, nil
}

// ListRuns returns one durable eval-run page with lightweight rows.
func (s *EvalRunStore) ListRuns(ctx context.Context, filter evalsvc.RunListFilter) (evalsvc.RunListPage, error) {
	const query = `
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM eval_runs
WHERE tenant_id = $1
  AND ($2 = '' OR dataset_id = $2)
  AND ($3 = '' OR status = $3)
ORDER BY updated_at DESC, id DESC
LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(ctx, query, filter.TenantID, filter.DatasetID, filter.Status, filter.Limit+1, filter.Offset)
	if err != nil {
		return evalsvc.RunListPage{}, fmt.Errorf("list eval runs: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalRun, 0, filter.Limit+1)
	for rows.Next() {
		var item evalsvc.EvalRun
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.DatasetID,
			&item.DatasetName,
			&item.DatasetItemCount,
			&item.Status,
			&item.CreatedBy,
			&item.ErrorReason,
			&item.CreatedAt,
			&item.UpdatedAt,
			&startedAt,
			&finishedAt,
		); err != nil {
			return evalsvc.RunListPage{}, fmt.Errorf("scan eval run: %w", err)
		}
		if startedAt.Valid {
			item.StartedAt = startedAt.Time
		}
		if finishedAt.Valid {
			item.FinishedAt = finishedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return evalsvc.RunListPage{}, fmt.Errorf("iterate eval runs: %w", err)
	}

	page := evalsvc.RunListPage{Runs: items}
	if len(items) > filter.Limit {
		page.HasMore = true
		page.NextOffset = filter.Offset + filter.Limit
		page.Runs = append([]evalsvc.EvalRun(nil), items[:filter.Limit]...)
	}
	return page, nil
}

// ClaimQueuedRuns marks queued eval runs as running and returns the claimed rows.
func (s *EvalRunStore) ClaimQueuedRuns(ctx context.Context, limit int, startedAt time.Time) ([]evalsvc.EvalRun, error) {
	const query = `
WITH claimed AS (
    SELECT id
    FROM eval_runs
    WHERE status = $1
    ORDER BY created_at ASC, id ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
),
updated AS (
    UPDATE eval_runs r
    SET status = $3,
        error_reason = '',
        updated_at = $4,
        started_at = COALESCE(r.started_at, $4)
    FROM claimed
    WHERE r.id = claimed.id
    RETURNING
        r.id,
        r.tenant_id,
        r.dataset_id,
        r.dataset_name,
        r.dataset_item_count,
        r.status,
        r.created_by,
        r.error_reason,
        r.created_at,
        r.updated_at,
        r.started_at,
        r.finished_at
)
SELECT
    id,
    tenant_id,
    dataset_id,
    dataset_name,
    dataset_item_count,
    status,
    created_by,
    error_reason,
    created_at,
    updated_at,
    started_at,
    finished_at
FROM updated
ORDER BY created_at ASC, id ASC`

	rows, err := s.pool.Query(ctx, query, evalsvc.RunStatusQueued, limit, evalsvc.RunStatusRunning, startedAt)
	if err != nil {
		return nil, fmt.Errorf("claim eval runs: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalRun, 0, limit)
	for rows.Next() {
		item, err := scanEvalRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed eval runs: %w", err)
	}

	return items, nil
}

// UpdateRun updates one durable eval run row.
func (s *EvalRunStore) UpdateRun(ctx context.Context, item evalsvc.EvalRun) (evalsvc.EvalRun, error) {
	const query = `
UPDATE eval_runs
SET tenant_id = $2,
    dataset_id = $3,
    dataset_name = $4,
    dataset_item_count = $5,
    status = $6,
    created_by = $7,
    error_reason = $8,
    created_at = $9,
    updated_at = $10,
    started_at = $11,
    finished_at = $12
WHERE id = $1`

	commandTag, err := s.pool.Exec(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.DatasetID,
		item.DatasetName,
		item.DatasetItemCount,
		item.Status,
		item.CreatedBy,
		item.ErrorReason,
		item.CreatedAt,
		item.UpdatedAt,
		nullTime(item.StartedAt),
		nullTime(item.FinishedAt),
	)
	if err != nil {
		return evalsvc.EvalRun{}, fmt.Errorf("update eval run: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return evalsvc.EvalRun{}, evalsvc.ErrEvalRunNotFound
	}

	return s.GetRun(ctx, item.ID)
}

func scanEvalRun(scanner interface {
	Scan(dest ...any) error
}) (evalsvc.EvalRun, error) {
	var item evalsvc.EvalRun
	var startedAt, finishedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.DatasetID,
		&item.DatasetName,
		&item.DatasetItemCount,
		&item.Status,
		&item.CreatedBy,
		&item.ErrorReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return evalsvc.EvalRun{}, fmt.Errorf("scan eval run: %w", err)
	}
	if startedAt.Valid {
		item.StartedAt = startedAt.Time
	}
	if finishedAt.Valid {
		item.FinishedAt = finishedAt.Time
	}
	return item, nil
}

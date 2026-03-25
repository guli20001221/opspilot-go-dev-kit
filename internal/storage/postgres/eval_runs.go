package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

// EvalDatasetStore persists durable eval datasets in PostgreSQL.
type EvalDatasetStore struct {
	pool *pgxpool.Pool
}

// NewEvalDatasetStore constructs the eval dataset repository.
func NewEvalDatasetStore(pool *pgxpool.Pool) *EvalDatasetStore {
	return &EvalDatasetStore{pool: pool}
}

// CreateDataset inserts one durable dataset draft plus its eval-case memberships.
func (s *EvalDatasetStore) CreateDataset(ctx context.Context, item evalsvc.EvalDataset) (evalsvc.EvalDataset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("begin eval dataset tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const datasetQuery = `
INSERT INTO eval_datasets (
    id,
    tenant_id,
    name,
    description,
    status,
    created_by,
    created_at,
    updated_at,
    published_by,
    published_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NULL, NULL
)`
	if _, err := tx.Exec(
		ctx,
		datasetQuery,
		item.ID,
		item.TenantID,
		item.Name,
		item.Description,
		item.Status,
		item.CreatedBy,
		item.CreatedAt,
		item.UpdatedAt,
	); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("insert eval dataset: %w", err)
	}

	const itemQuery = `
INSERT INTO eval_dataset_items (
    dataset_id,
    eval_case_id,
    position,
    created_at
) VALUES (
    $1, $2, $3, $4
)`
	for i, member := range item.Items {
		if _, err := tx.Exec(ctx, itemQuery, item.ID, member.EvalCaseID, i, item.CreatedAt); err != nil {
			return evalsvc.EvalDataset{}, fmt.Errorf("insert eval dataset item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("commit eval dataset tx: %w", err)
	}

	return s.GetDataset(ctx, item.ID)
}

// AddDatasetItem appends one durable eval-case membership into an existing draft dataset.
func (s *EvalDatasetStore) AddDatasetItem(ctx context.Context, datasetID string, item evalsvc.EvalDatasetItem, updatedAt time.Time) (evalsvc.EvalDataset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("begin eval dataset item tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const lockQuery = `
SELECT status
FROM eval_datasets
WHERE id = $1
FOR UPDATE`
	var status string
	if err := tx.QueryRow(ctx, lockQuery, datasetID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
		}
		return evalsvc.EvalDataset{}, fmt.Errorf("lock eval dataset: %w", err)
	}
	if status != evalsvc.DatasetStatusDraft {
		return evalsvc.EvalDataset{}, evalsvc.ErrInvalidEvalDatasetState
	}

	const insertQuery = `
INSERT INTO eval_dataset_items (
    dataset_id,
    eval_case_id,
    position,
    created_at
) VALUES (
    $1,
    $2,
    (
        SELECT COALESCE(MAX(position), -1) + 1
        FROM eval_dataset_items
        WHERE dataset_id = $1
    ),
    $3
)
ON CONFLICT (dataset_id, eval_case_id) DO NOTHING`
	commandTag, err := tx.Exec(ctx, insertQuery, datasetID, item.EvalCaseID, updatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return evalsvc.EvalDataset{}, evalsvc.ErrEvalCaseNotFound
		}
		return evalsvc.EvalDataset{}, fmt.Errorf("insert eval dataset item: %w", err)
	}
	if commandTag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `UPDATE eval_datasets SET updated_at = $2 WHERE id = $1`, datasetID, updatedAt); err != nil {
			return evalsvc.EvalDataset{}, fmt.Errorf("update eval dataset timestamp: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("commit eval dataset item tx: %w", err)
	}

	return s.GetDataset(ctx, datasetID)
}

// PublishDataset freezes one durable eval dataset draft into an immutable published baseline.
func (s *EvalDatasetStore) PublishDataset(ctx context.Context, datasetID string, publishedBy string, publishedAt time.Time) (evalsvc.EvalDataset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("begin eval dataset publish tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const lockQuery = `
SELECT status
FROM eval_datasets
WHERE id = $1
FOR UPDATE`
	var status string
	if err := tx.QueryRow(ctx, lockQuery, datasetID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
		}
		return evalsvc.EvalDataset{}, fmt.Errorf("lock eval dataset for publish: %w", err)
	}
	if status != evalsvc.DatasetStatusDraft {
		return evalsvc.EvalDataset{}, evalsvc.ErrInvalidEvalDatasetState
	}

	const publishQuery = `
UPDATE eval_datasets
SET status = $2,
    published_by = $3,
    published_at = $4,
    updated_at = $4
WHERE id = $1`
	if _, err := tx.Exec(ctx, publishQuery, datasetID, evalsvc.DatasetStatusPublished, publishedBy, publishedAt); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("publish eval dataset: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("commit eval dataset publish tx: %w", err)
	}

	return s.GetDataset(ctx, datasetID)
}

// GetDataset loads one durable eval dataset plus its items.
func (s *EvalDatasetStore) GetDataset(ctx context.Context, datasetID string) (evalsvc.EvalDataset, error) {
	const datasetQuery = `
SELECT
    id,
    tenant_id,
    name,
    description,
    status,
    created_by,
    created_at,
    updated_at,
    COALESCE(published_by, ''),
    published_at
FROM eval_datasets
WHERE id = $1`

	var item evalsvc.EvalDataset
	var publishedAt sql.NullTime
	if err := s.pool.QueryRow(ctx, datasetQuery, datasetID).Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Status,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.PublishedBy,
		&publishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalDataset{}, evalsvc.ErrEvalDatasetNotFound
		}
		return evalsvc.EvalDataset{}, fmt.Errorf("scan eval dataset: %w", err)
	}
	if publishedAt.Valid {
		item.PublishedAt = publishedAt.Time
	}

	const itemsQuery = `
SELECT
    e.id,
    e.title,
    e.source_case_id,
    COALESCE(e.source_task_id, ''),
    COALESCE(e.source_report_id, ''),
    e.trace_id,
    COALESCE(e.version_id, '')
FROM eval_dataset_items i
JOIN eval_cases e ON e.id = i.eval_case_id
WHERE i.dataset_id = $1
  AND e.tenant_id = $2
ORDER BY i.position ASC`

	rows, err := s.pool.Query(ctx, itemsQuery, datasetID, item.TenantID)
	if err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("query eval dataset items: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalDatasetItem, 0)
	for rows.Next() {
		var member evalsvc.EvalDatasetItem
		if err := rows.Scan(
			&member.EvalCaseID,
			&member.Title,
			&member.SourceCaseID,
			&member.SourceTaskID,
			&member.SourceReportID,
			&member.TraceID,
			&member.VersionID,
		); err != nil {
			return evalsvc.EvalDataset{}, fmt.Errorf("scan eval dataset item: %w", err)
		}
		items = append(items, member)
	}
	if err := rows.Err(); err != nil {
		return evalsvc.EvalDataset{}, fmt.Errorf("iterate eval dataset items: %w", err)
	}
	item.Items = items

	return item, nil
}

// ListDatasets returns one durable eval-dataset page with lightweight rows.
func (s *EvalDatasetStore) ListDatasets(ctx context.Context, filter evalsvc.DatasetListFilter) (evalsvc.DatasetListPage, error) {
	const query = `
SELECT
    d.id,
    d.tenant_id,
    d.name,
    d.status,
    d.created_by,
    d.created_at,
    d.updated_at,
    COUNT(e.id)::INT AS item_count
FROM eval_datasets d
LEFT JOIN eval_dataset_items i ON i.dataset_id = d.id
LEFT JOIN eval_cases e ON e.id = i.eval_case_id AND e.tenant_id = d.tenant_id
WHERE d.tenant_id = $1
  AND ($2 = '' OR d.status = $2)
  AND ($3 = '' OR d.created_by = $3)
GROUP BY
    d.id,
    d.tenant_id,
    d.name,
    d.status,
    d.created_by,
    d.created_at,
    d.updated_at
ORDER BY d.updated_at DESC, d.id DESC
LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(
		ctx,
		query,
		filter.TenantID,
		filter.Status,
		filter.CreatedBy,
		filter.Limit+1,
		filter.Offset,
	)
	if err != nil {
		return evalsvc.DatasetListPage{}, fmt.Errorf("list eval datasets: %w", err)
	}
	defer rows.Close()

	items := make([]evalsvc.EvalDatasetSummary, 0, filter.Limit+1)
	for rows.Next() {
		var item evalsvc.EvalDatasetSummary
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.Name,
			&item.Status,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ItemCount,
		); err != nil {
			return evalsvc.DatasetListPage{}, fmt.Errorf("scan eval dataset summary: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return evalsvc.DatasetListPage{}, fmt.Errorf("iterate eval datasets: %w", err)
	}

	page := evalsvc.DatasetListPage{Datasets: items}
	if len(items) > filter.Limit {
		page.HasMore = true
		page.NextOffset = filter.Offset + filter.Limit
		page.Datasets = append([]evalsvc.EvalDatasetSummary(nil), items[:filter.Limit]...)
	}

	return page, nil
}

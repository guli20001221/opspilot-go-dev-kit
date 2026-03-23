package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	casesvc "opspilot-go/internal/case"
)

// CaseStore persists operator-managed cases in PostgreSQL.
type CaseStore struct {
	pool *pgxpool.Pool
}

type caseQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewCaseStore constructs the case repository.
func NewCaseStore(pool *pgxpool.Pool) *CaseStore {
	return &CaseStore{pool: pool}
}

// Save inserts or updates a durable case record.
func (s *CaseStore) Save(ctx context.Context, item casesvc.Case) (casesvc.Case, error) {
	const query = `
INSERT INTO cases (
    id,
    tenant_id,
    status,
    title,
    summary,
    source_task_id,
    source_report_id,
    created_by,
    closed_by,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10, $11
)
ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    status = EXCLUDED.status,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    source_task_id = EXCLUDED.source_task_id,
    source_report_id = EXCLUDED.source_report_id,
    created_by = EXCLUDED.created_by,
    closed_by = EXCLUDED.closed_by,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at
RETURNING
    id,
    tenant_id,
    status,
    title,
    summary,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    created_by,
    closed_by,
    created_at,
    updated_at`

	row := s.pool.QueryRow(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.Status,
		item.Title,
		item.Summary,
		item.SourceTaskID,
		item.SourceReportID,
		item.CreatedBy,
		item.ClosedBy,
		item.CreatedAt,
		item.UpdatedAt,
	)

	return scanCase(row)
}

// Get loads a case by ID.
func (s *CaseStore) Get(ctx context.Context, caseID string) (casesvc.Case, error) {
	const query = `
SELECT
    id,
    tenant_id,
    status,
    title,
    summary,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    created_by,
    closed_by,
    created_at,
    updated_at
FROM cases
WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, caseID)
	got, err := scanCase(row)
	if err != nil {
		if errors.Is(err, casesvc.ErrCaseNotFound) {
			return casesvc.Case{}, fmt.Errorf("%w: %s", casesvc.ErrCaseNotFound, caseID)
		}
		return casesvc.Case{}, err
	}

	return got, nil
}

// List returns filtered case rows for operator-facing list views.
func (s *CaseStore) List(ctx context.Context, filter casesvc.ListFilter) (casesvc.ListPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	const query = `
SELECT
    id,
    tenant_id,
    status,
    title,
    summary,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    created_by,
    closed_by,
    created_at,
    updated_at
FROM cases
WHERE ($1 = '' OR tenant_id = $1)
  AND ($2 = '' OR status = $2)
  AND ($3 = '' OR source_task_id = $3)
  AND ($4 = '' OR source_report_id = $4)
ORDER BY updated_at DESC, created_at DESC, id DESC
LIMIT $5 OFFSET $6`

	rows, err := s.pool.Query(ctx, query, filter.TenantID, filter.Status, filter.SourceTaskID, filter.SourceReportID, limit+1, offset)
	if err != nil {
		return casesvc.ListPage{}, fmt.Errorf("select cases: %w", err)
	}
	defer rows.Close()

	var items []casesvc.Case
	for rows.Next() {
		item, err := scanCase(rows)
		if err != nil {
			return casesvc.ListPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return casesvc.ListPage{}, fmt.Errorf("iterate cases: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	page := casesvc.ListPage{
		Cases:   items,
		HasMore: hasMore,
	}
	if hasMore {
		page.NextOffset = offset + len(items)
	}

	return page, nil
}

// Close atomically transitions an open case to closed.
func (s *CaseStore) Close(ctx context.Context, caseID string, closedBy string, closedAt time.Time) (casesvc.Case, error) {
	const query = `
UPDATE cases
SET status = $2,
    closed_by = $3,
    updated_at = $4
WHERE id = $1
  AND status <> $2
RETURNING
    id,
    tenant_id,
    status,
    title,
    summary,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    created_by,
    closed_by,
    created_at,
    updated_at`

	row := s.pool.QueryRow(ctx, query, caseID, casesvc.StatusClosed, closedBy, closedAt)
	closed, err := scanCase(row)
	if err == nil {
		return closed, nil
	}
	if !errors.Is(err, casesvc.ErrCaseNotFound) {
		return casesvc.Case{}, err
	}

	existing, getErr := s.Get(ctx, caseID)
	if getErr != nil {
		return casesvc.Case{}, getErr
	}
	if existing.Status == casesvc.StatusClosed {
		return casesvc.Case{}, casesvc.ErrInvalidCaseState
	}

	return casesvc.Case{}, err
}

func scanCase(row caseQuerierRow) (casesvc.Case, error) {
	var item casesvc.Case
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Status,
		&item.Title,
		&item.Summary,
		&item.SourceTaskID,
		&item.SourceReportID,
		&item.CreatedBy,
		&item.ClosedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return casesvc.Case{}, casesvc.ErrCaseNotFound
		}
		return casesvc.Case{}, fmt.Errorf("scan case: %w", err)
	}

	return item, nil
}

type caseQuerierRow interface {
	Scan(dest ...any) error
}

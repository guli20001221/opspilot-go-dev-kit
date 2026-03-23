package postgres

import (
	"context"
	"errors"
	"fmt"

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
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10
)
ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    status = EXCLUDED.status,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    source_task_id = EXCLUDED.source_task_id,
    source_report_id = EXCLUDED.source_report_id,
    created_by = EXCLUDED.created_by,
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

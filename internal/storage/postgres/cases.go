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

type caseNoteQuerierRow interface {
	Scan(dest ...any) error
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
    assigned_to,
    assigned_at,
    closed_by,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    status = EXCLUDED.status,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    source_task_id = EXCLUDED.source_task_id,
    source_report_id = EXCLUDED.source_report_id,
    created_by = EXCLUDED.created_by,
    assigned_to = EXCLUDED.assigned_to,
    assigned_at = EXCLUDED.assigned_at,
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
    assigned_to,
    assigned_at,
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
		item.AssignedTo,
		nullTime(item.AssignedAt),
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
    assigned_to,
    assigned_at,
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
    assigned_to,
    assigned_at,
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

// AppendNote stores an append-only operator note for a case.
func (s *CaseStore) AppendNote(ctx context.Context, note casesvc.Note) (casesvc.Note, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return casesvc.Note{}, fmt.Errorf("begin case note tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateCaseQuery = `
UPDATE cases
SET updated_at = $2
WHERE id = $1
  AND tenant_id = $3
RETURNING id`

	var caseID string
	if err := tx.QueryRow(ctx, updateCaseQuery, note.CaseID, note.CreatedAt, note.TenantID).Scan(&caseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return casesvc.Note{}, fmt.Errorf("%w: %s", casesvc.ErrCaseNotFound, note.CaseID)
		}
		return casesvc.Note{}, fmt.Errorf("update case recency: %w", err)
	}

	const query = `
INSERT INTO case_notes (
    id,
    tenant_id,
    case_id,
    body,
    created_by,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING
    id,
    tenant_id,
    case_id,
    body,
    created_by,
    created_at`

	row := tx.QueryRow(ctx, query, note.ID, note.TenantID, note.CaseID, note.Body, note.CreatedBy, note.CreatedAt)
	item, err := scanCaseNote(row)
	if err != nil {
		return casesvc.Note{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return casesvc.Note{}, fmt.Errorf("commit case note tx: %w", err)
	}

	return item, nil
}

// ListNotes returns recent notes for a case in newest-first order.
func (s *CaseStore) ListNotes(ctx context.Context, caseID string, limit int) ([]casesvc.Note, error) {
	if limit <= 0 {
		limit = 20
	}

	const query = `
SELECT
    id,
    tenant_id,
    case_id,
    body,
    created_by,
    created_at
FROM case_notes
WHERE case_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`

	rows, err := s.pool.Query(ctx, query, caseID, limit)
	if err != nil {
		return nil, fmt.Errorf("select case notes: %w", err)
	}
	defer rows.Close()

	items := make([]casesvc.Note, 0, limit)
	for rows.Next() {
		item, err := scanCaseNote(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case notes: %w", err)
	}

	return items, nil
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

// Assign atomically assigns an open case using optimistic concurrency on updated_at.
func (s *CaseStore) Assign(ctx context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (casesvc.Case, error) {
	const query = `
UPDATE cases
SET assigned_to = $2,
    assigned_at = $3,
    updated_at = $3
WHERE id = $1
  AND status = $4
  AND updated_at = $5
RETURNING
    id,
    tenant_id,
    status,
    title,
    summary,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    created_by,
    assigned_to,
    assigned_at,
    closed_by,
    created_at,
    updated_at`

	row := s.pool.QueryRow(ctx, query, caseID, assignedTo, assignedAt, casesvc.StatusOpen, expectedUpdatedAt)
	assigned, err := scanCase(row)
	if err == nil {
		return assigned, nil
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
	if !existing.UpdatedAt.Equal(expectedUpdatedAt) {
		return casesvc.Case{}, casesvc.ErrCaseConflict
	}

	return casesvc.Case{}, err
}

func scanCase(row caseQuerierRow) (casesvc.Case, error) {
	var item casesvc.Case
	var assignedAt *time.Time
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Status,
		&item.Title,
		&item.Summary,
		&item.SourceTaskID,
		&item.SourceReportID,
		&item.CreatedBy,
		&item.AssignedTo,
		&assignedAt,
		&item.ClosedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return casesvc.Case{}, casesvc.ErrCaseNotFound
		}
		return casesvc.Case{}, fmt.Errorf("scan case: %w", err)
	}
	if assignedAt != nil {
		item.AssignedAt = *assignedAt
	}

	return item, nil
}

func scanCaseNote(row caseNoteQuerierRow) (casesvc.Note, error) {
	var item casesvc.Note
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.CaseID,
		&item.Body,
		&item.CreatedBy,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return casesvc.Note{}, casesvc.ErrCaseNotFound
		}
		return casesvc.Note{}, fmt.Errorf("scan case note: %w", err)
	}

	return item, nil
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value
}

type caseQuerierRow interface {
	Scan(dest ...any) error
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

// EvalCaseStore persists durable eval case records in PostgreSQL.
type EvalCaseStore struct {
	pool *pgxpool.Pool
}

// NewEvalCaseStore constructs the eval case repository.
func NewEvalCaseStore(pool *pgxpool.Pool) *EvalCaseStore {
	return &EvalCaseStore{pool: pool}
}

// Save inserts or updates a durable eval case record.
func (s *EvalCaseStore) Save(ctx context.Context, item evalsvc.EvalCase) (evalsvc.EvalCase, error) {
	const query = `
INSERT INTO eval_cases (
    id,
    tenant_id,
    source_case_id,
    source_task_id,
    source_report_id,
    trace_id,
    version_id,
    title,
    summary,
    operator_note,
    created_by,
    created_at
) VALUES (
    $1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, NULLIF($7, ''), $8, $9, $10, $11, $12
)
RETURNING
    id,
    tenant_id,
    source_case_id,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    trace_id,
    COALESCE(version_id, ''),
    title,
    summary,
    operator_note,
    created_by,
    created_at`

	row := s.pool.QueryRow(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.SourceCaseID,
		item.SourceTaskID,
		item.SourceReportID,
		item.TraceID,
		item.VersionID,
		item.Title,
		item.Summary,
		item.OperatorNote,
		item.CreatedBy,
		item.CreatedAt,
	)

	saved, err := scanEvalCase(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "eval_cases_source_case_id_key" {
			return evalsvc.EvalCase{}, fmt.Errorf("%w: %s", evalsvc.ErrEvalCaseExists, item.SourceCaseID)
		}
		return evalsvc.EvalCase{}, err
	}

	return saved, nil
}

// Get loads an eval case by ID.
func (s *EvalCaseStore) Get(ctx context.Context, evalCaseID string) (evalsvc.EvalCase, error) {
	const query = `
SELECT
    id,
    tenant_id,
    source_case_id,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    trace_id,
    COALESCE(version_id, ''),
    title,
    summary,
    operator_note,
    created_by,
    created_at
FROM eval_cases
WHERE id = $1`

	return scanEvalCase(s.pool.QueryRow(ctx, query, evalCaseID))
}

// GetBySourceCase loads an eval case by source case lineage.
func (s *EvalCaseStore) GetBySourceCase(ctx context.Context, sourceCaseID string) (evalsvc.EvalCase, error) {
	const query = `
SELECT
    id,
    tenant_id,
    source_case_id,
    COALESCE(source_task_id, ''),
    COALESCE(source_report_id, ''),
    trace_id,
    COALESCE(version_id, ''),
    title,
    summary,
    operator_note,
    created_by,
    created_at
FROM eval_cases
WHERE source_case_id = $1`

	return scanEvalCase(s.pool.QueryRow(ctx, query, sourceCaseID))
}

func scanEvalCase(row pgx.Row) (evalsvc.EvalCase, error) {
	var item evalsvc.EvalCase
	var versionID pgtype.Text
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.SourceCaseID,
		&item.SourceTaskID,
		&item.SourceReportID,
		&item.TraceID,
		&versionID,
		&item.Title,
		&item.Summary,
		&item.OperatorNote,
		&item.CreatedBy,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalCase{}, evalsvc.ErrEvalCaseNotFound
		}
		return evalsvc.EvalCase{}, fmt.Errorf("scan eval case: %w", err)
	}
	if versionID.Valid {
		item.VersionID = versionID.String
	}

	return item, nil
}

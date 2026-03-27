package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

// EvalReportStore persists durable eval-report artifacts in PostgreSQL.
type EvalReportStore struct {
	pool *pgxpool.Pool
}

type evalReportQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewEvalReportStore constructs the eval-report repository.
func NewEvalReportStore(pool *pgxpool.Pool) *EvalReportStore {
	return &EvalReportStore{pool: pool}
}

// SaveEvalReport inserts or updates one durable eval report.
func (s *EvalReportStore) SaveEvalReport(ctx context.Context, item evalsvc.EvalReport) (evalsvc.EvalReport, error) {
	const query = `
INSERT INTO eval_reports (
    id,
    tenant_id,
    run_id,
    dataset_id,
    dataset_name,
    run_status,
    status,
    summary,
    total_items,
    recorded_results,
    passed_items,
    failed_items,
    missing_results,
    average_score,
    judge_version,
    metadata_json,
    bad_cases_json,
    created_at,
    updated_at,
    ready_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
)
ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    run_id = EXCLUDED.run_id,
    dataset_id = EXCLUDED.dataset_id,
    dataset_name = EXCLUDED.dataset_name,
    run_status = EXCLUDED.run_status,
    status = EXCLUDED.status,
    summary = EXCLUDED.summary,
    total_items = EXCLUDED.total_items,
    recorded_results = EXCLUDED.recorded_results,
    passed_items = EXCLUDED.passed_items,
    failed_items = EXCLUDED.failed_items,
    missing_results = EXCLUDED.missing_results,
    average_score = EXCLUDED.average_score,
    judge_version = EXCLUDED.judge_version,
    metadata_json = EXCLUDED.metadata_json,
    bad_cases_json = EXCLUDED.bad_cases_json,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at,
    ready_at = EXCLUDED.ready_at
RETURNING
    id,
    tenant_id,
    run_id,
    dataset_id,
    dataset_name,
    run_status,
    status,
    summary,
    total_items,
    recorded_results,
    passed_items,
    failed_items,
    missing_results,
    average_score,
    judge_version,
    metadata_json,
    bad_cases_json,
    created_at,
    updated_at,
    ready_at`

	badCasesJSON, err := json.Marshal(item.BadCases)
	if err != nil {
		return evalsvc.EvalReport{}, fmt.Errorf("marshal eval report bad cases: %w", err)
	}

	row := s.pool.QueryRow(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.RunID,
		item.DatasetID,
		item.DatasetName,
		item.RunStatus,
		item.Status,
		item.Summary,
		item.TotalItems,
		item.RecordedResults,
		item.PassedItems,
		item.FailedItems,
		item.MissingResults,
		item.AverageScore,
		item.JudgeVersion,
		item.MetadataJSON,
		badCasesJSON,
		item.CreatedAt,
		item.UpdatedAt,
		item.ReadyAt,
	)

	return scanEvalReport(row)
}

// GetEvalReport loads one durable eval report by ID.
func (s *EvalReportStore) GetEvalReport(ctx context.Context, reportID string) (evalsvc.EvalReport, error) {
	const query = `
SELECT
    id,
    tenant_id,
    run_id,
    dataset_id,
    dataset_name,
    run_status,
    status,
    summary,
    total_items,
    recorded_results,
    passed_items,
    failed_items,
    missing_results,
    average_score,
    judge_version,
    metadata_json,
    bad_cases_json,
    created_at,
    updated_at,
    ready_at
FROM eval_reports
WHERE id = $1`

	return scanEvalReport(s.pool.QueryRow(ctx, query, reportID))
}

func scanEvalReport(row pgx.Row) (evalsvc.EvalReport, error) {
	var item evalsvc.EvalReport
	var badCasesJSON json.RawMessage
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.RunID,
		&item.DatasetID,
		&item.DatasetName,
		&item.RunStatus,
		&item.Status,
		&item.Summary,
		&item.TotalItems,
		&item.RecordedResults,
		&item.PassedItems,
		&item.FailedItems,
		&item.MissingResults,
		&item.AverageScore,
		&item.JudgeVersion,
		&item.MetadataJSON,
		&badCasesJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ReadyAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return evalsvc.EvalReport{}, evalsvc.ErrEvalReportNotFound
		}
		return evalsvc.EvalReport{}, fmt.Errorf("scan eval report: %w", err)
	}
	if len(badCasesJSON) > 0 {
		if err := json.Unmarshal(badCasesJSON, &item.BadCases); err != nil {
			return evalsvc.EvalReport{}, fmt.Errorf("decode eval report bad cases: %w", err)
		}
	}

	return item, nil
}

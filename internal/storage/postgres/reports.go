package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/report"
)

// ReportStore persists report read models in PostgreSQL.
type ReportStore struct {
	pool *pgxpool.Pool
}

// NewReportStore constructs the report repository.
func NewReportStore(pool *pgxpool.Pool) *ReportStore {
	return &ReportStore{pool: pool}
}

// Save inserts or updates a durable report record.
func (s *ReportStore) Save(ctx context.Context, item report.Report) (report.Report, error) {
	const query = `
INSERT INTO reports (
    id,
    tenant_id,
    source_task_id,
    report_type,
    status,
    title,
    summary,
    content_uri,
    metadata_json,
    created_by,
    created_at,
    ready_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    source_task_id = EXCLUDED.source_task_id,
    report_type = EXCLUDED.report_type,
    status = EXCLUDED.status,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    content_uri = EXCLUDED.content_uri,
    metadata_json = EXCLUDED.metadata_json,
    created_by = EXCLUDED.created_by,
    created_at = EXCLUDED.created_at,
    ready_at = EXCLUDED.ready_at
RETURNING
    id,
    tenant_id,
    source_task_id,
    report_type,
    status,
    title,
    summary,
    content_uri,
    metadata_json,
    created_by,
    created_at,
    ready_at`

	row := s.pool.QueryRow(
		ctx,
		query,
		item.ID,
		item.TenantID,
		item.SourceTaskID,
		item.ReportType,
		item.Status,
		item.Title,
		item.Summary,
		item.ContentURI,
		item.MetadataJSON,
		item.CreatedBy,
		item.CreatedAt,
		item.ReadyAt,
	)

	saved, err := scanReport(row)
	if err != nil {
		return report.Report{}, err
	}

	return saved, nil
}

// Get loads a report by ID.
func (s *ReportStore) Get(ctx context.Context, reportID string) (report.Report, error) {
	const query = `
SELECT
    id,
    tenant_id,
    source_task_id,
    report_type,
    status,
    title,
    summary,
    content_uri,
    metadata_json,
    created_by,
    created_at,
    ready_at
FROM reports
WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, reportID)
	got, err := scanReport(row)
	if err != nil {
		if errors.Is(err, report.ErrReportNotFound) {
			return report.Report{}, fmt.Errorf("%w: %s", report.ErrReportNotFound, reportID)
		}
		return report.Report{}, err
	}

	return got, nil
}

func scanReport(row pgx.Row) (report.Report, error) {
	var item report.Report
	var metadata json.RawMessage
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.SourceTaskID,
		&item.ReportType,
		&item.Status,
		&item.Title,
		&item.Summary,
		&item.ContentURI,
		&metadata,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.ReadyAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return report.Report{}, report.ErrReportNotFound
		}

		return report.Report{}, fmt.Errorf("scan report: %w", err)
	}
	item.MetadataJSON = metadata

	return item, nil
}

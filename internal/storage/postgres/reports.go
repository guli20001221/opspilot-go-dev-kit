package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

// ReportStore persists report read models in PostgreSQL.
type ReportStore struct {
	pool *pgxpool.Pool
}

type reportQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewReportStore constructs the report repository.
func NewReportStore(pool *pgxpool.Pool) *ReportStore {
	return &ReportStore{pool: pool}
}

// Save inserts or updates a durable report record.
func (s *ReportStore) Save(ctx context.Context, item report.Report) (report.Report, error) {
	saved, err := s.save(ctx, s.pool, item)
	if err != nil {
		return report.Report{}, err
	}

	return saved, nil
}

// FinalizeSucceededTaskWithReport atomically writes the durable report and the
// successful workflow task transition.
func (s *ReportStore) FinalizeSucceededTaskWithReport(ctx context.Context, task workflow.Task, event workflow.AuditEvent, item report.Report) (report.Report, workflow.Task, error) {
	var saved report.Report
	var updated workflow.Task
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return report.Report{}, workflow.Task{}, fmt.Errorf("begin report finalization transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(ctx, updateTaskQuery, task.ID, task.Status, task.ErrorReason, task.AuditRef, task.UpdatedAt)
	updated, err = scanTask(row)
	if err != nil {
		return report.Report{}, workflow.Task{}, err
	}
	if _, err := (&WorkflowTaskStore{}).appendTaskEvent(ctx, tx, event); err != nil {
		return report.Report{}, workflow.Task{}, err
	}
	saved, err = s.save(ctx, tx, item)
	if err != nil {
		return report.Report{}, workflow.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return report.Report{}, workflow.Task{}, fmt.Errorf("commit report finalization transaction: %w", err)
	}

	return saved, updated, nil
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

// List returns a durable report page.
func (s *ReportStore) List(ctx context.Context, filter report.ListFilter) (report.ListPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var (
		where []string
		args  []any
	)
	if filter.TenantID != "" {
		args = append(args, filter.TenantID)
		where = append(where, fmt.Sprintf("tenant_id = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.ReportType != "" {
		args = append(args, filter.ReportType)
		where = append(where, fmt.Sprintf("report_type = $%d", len(args)))
	}
	if filter.SourceTaskID != "" {
		args = append(args, filter.SourceTaskID)
		where = append(where, fmt.Sprintf("source_task_id = $%d", len(args)))
	}

	query := `
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
FROM reports`
	if len(where) > 0 {
		query += "\nWHERE " + strings.Join(where, " AND ")
	}

	args = append(args, limit+1, filter.Offset)
	query += fmt.Sprintf(`
ORDER BY COALESCE(ready_at, created_at) DESC, created_at DESC, id DESC
LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return report.ListPage{}, fmt.Errorf("query reports: %w", err)
	}
	defer rows.Close()

	items := make([]report.Report, 0, limit+1)
	for rows.Next() {
		item, err := scanReport(rows)
		if err != nil {
			return report.ListPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return report.ListPage{}, fmt.Errorf("iterate reports: %w", err)
	}

	page := report.ListPage{Reports: items}
	if len(items) > limit {
		page.HasMore = true
		page.NextOffset = filter.Offset + limit
		page.Reports = items[:limit]
	}

	return page, nil
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

func (s *ReportStore) save(ctx context.Context, db reportQuerier, item report.Report) (report.Report, error) {
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

	row := db.QueryRow(
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

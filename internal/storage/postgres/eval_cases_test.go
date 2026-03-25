package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	evalsvc "opspilot-go/internal/eval"
)

func TestEvalCaseStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval_cases and lineage tables error = %v", err)
	}
	seedEvalLineage(t, ctx, pool)

	store := NewEvalCaseStore(pool)
	want := evalsvc.EvalCase{
		ID:             "eval-case-roundtrip",
		TenantID:       "tenant-eval",
		SourceCaseID:   "case-eval-1",
		SourceTaskID:   "task-eval-1",
		SourceReportID: "report-task-eval-1",
		TraceID:        "trace-eval-1",
		VersionID:      "version-skeleton-2026-03-24",
		Title:          "Investigate retrieval drift",
		Summary:        "Promoted for regression coverage.",
		OperatorNote:   "capture this failure for eval",
		CreatedBy:      "operator-1",
		CreatedAt:      time.Unix(1700013000, 0).UTC(),
	}

	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.SourceCaseID != want.SourceCaseID {
		t.Fatalf("SourceCaseID = %q, want %q", got.SourceCaseID, want.SourceCaseID)
	}
	if got.VersionID != want.VersionID {
		t.Fatalf("VersionID = %q, want %q", got.VersionID, want.VersionID)
	}

	bySource, err := store.GetBySourceCase(ctx, want.SourceCaseID)
	if err != nil {
		t.Fatalf("GetBySourceCase() error = %v", err)
	}
	if bySource.ID != want.ID {
		t.Fatalf("GetBySourceCase().ID = %q, want %q", bySource.ID, want.ID)
	}
}

func TestEvalCaseStoreListSupportsFiltersAndPagination(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval_cases and lineage tables error = %v", err)
	}
	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-a", "report-task-eval-a", "case-eval-a", "version-a", "trace-a", time.Unix(1700012900, 0).UTC())
	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-b", "report-task-eval-b", "case-eval-b", "version-b", "trace-b", time.Unix(1700012910, 0).UTC())
	seedEvalLineageCase(t, ctx, pool, "tenant-other", "task-eval-c", "report-task-eval-c", "case-eval-c", "version-b", "trace-c", time.Unix(1700012920, 0).UTC())

	store := NewEvalCaseStore(pool)
	for _, item := range []evalsvc.EvalCase{
		{
			ID:             "eval-case-a",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-a",
			SourceTaskID:   "task-eval-a",
			SourceReportID: "report-task-eval-a",
			TraceID:        "trace-a",
			VersionID:      "version-a",
			Title:          "Eval A",
			Summary:        "First",
			CreatedBy:      "operator-a",
			CreatedAt:      time.Unix(1700013000, 0).UTC(),
		},
		{
			ID:             "eval-case-b",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-b",
			SourceTaskID:   "task-eval-b",
			SourceReportID: "report-task-eval-b",
			TraceID:        "trace-b",
			VersionID:      "version-b",
			Title:          "Eval B",
			Summary:        "Second",
			CreatedBy:      "operator-b",
			CreatedAt:      time.Unix(1700013010, 0).UTC(),
		},
		{
			ID:             "eval-case-c",
			TenantID:       "tenant-other",
			SourceCaseID:   "case-eval-c",
			SourceTaskID:   "task-eval-c",
			SourceReportID: "report-task-eval-c",
			TraceID:        "trace-c",
			VersionID:      "version-b",
			Title:          "Eval C",
			Summary:        "Third",
			CreatedBy:      "operator-c",
			CreatedAt:      time.Unix(1700013020, 0).UTC(),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(ctx, evalsvc.ListFilter{
		TenantID:  "tenant-eval",
		VersionID: "version-b",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.EvalCases) != 1 || page.EvalCases[0].ID != "eval-case-b" {
		t.Fatalf("EvalCases = %#v, want eval-case-b", page.EvalCases)
	}

	page, err = store.List(ctx, evalsvc.ListFilter{
		TenantID: "tenant-eval",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("List(paginated) error = %v", err)
	}
	if len(page.EvalCases) != 1 || page.EvalCases[0].ID != "eval-case-b" {
		t.Fatalf("first page = %#v, want eval-case-b", page.EvalCases)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}
}

func seedEvalLineage(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-1", "report-task-eval-1", "case-eval-1", "version-skeleton-2026-03-24", "trace-eval-1", time.Unix(1700012900, 0).UTC())
}

func seedEvalLineageCase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID string, taskID string, reportID string, caseID string, versionID string, traceID string, createdAt time.Time) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
INSERT INTO workflow_tasks (
    id,
    request_id,
    tenant_id,
    session_id,
    task_type,
    status,
    reason,
    requires_approval,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    'report_generation',
    'succeeded',
    'workflow_required',
    false,
    $5,
    $5
)`, taskID, "req-"+taskID, tenantID, "session-"+taskID, createdAt); err != nil {
		t.Fatalf("insert workflow task error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO reports (
    id,
    tenant_id,
    source_task_id,
    version_id,
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
    $1,
    $2,
    $3,
    $4,
    'workflow_summary',
    'ready',
    $5,
    'generated report',
    '',
    $6,
    'worker',
    $7,
    $7
)`, reportID, tenantID, taskID, versionID, "Report for "+taskID, fmt.Sprintf(`{"trace_id":"%s"}`, traceID), createdAt.Add(50*time.Second)); err != nil {
		t.Fatalf("insert report error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
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
    $1,
    $2,
    'open',
    $3,
    'Promoted for regression coverage.',
    $4,
    $5,
    'operator-1',
    '',
    NULL,
    '',
    $6,
    $6
)`, caseID, tenantID, "Investigate "+taskID, taskID, reportID, createdAt.Add(90*time.Second)); err != nil {
		t.Fatalf("insert case error = %v", err)
	}
}

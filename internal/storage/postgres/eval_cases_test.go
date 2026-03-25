package postgres

import (
	"context"
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

func seedEvalLineage(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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
    'task-eval-1',
    'req-eval-1',
    'tenant-eval',
    'session-eval-1',
    'report_generation',
    'succeeded',
    'workflow_required',
    false,
    $1,
    $1
)`, time.Unix(1700012900, 0).UTC()); err != nil {
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
    'report-task-eval-1',
    'tenant-eval',
    'task-eval-1',
    'version-skeleton-2026-03-24',
    'workflow_summary',
    'ready',
    'Report for task-eval-1',
    'generated report',
    '',
    '{"trace_id":"trace-eval-1"}',
    'worker',
    $1,
    $1
)`, time.Unix(1700012950, 0).UTC()); err != nil {
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
    'case-eval-1',
    'tenant-eval',
    'open',
    'Investigate retrieval drift',
    'Promoted for regression coverage.',
    'task-eval-1',
    'report-task-eval-1',
    'operator-1',
    '',
    NULL,
    '',
    $1,
    $1
)`, time.Unix(1700012990, 0).UTC()); err != nil {
		t.Fatalf("insert case error = %v", err)
	}
}

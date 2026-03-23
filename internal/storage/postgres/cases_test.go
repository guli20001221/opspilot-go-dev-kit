package postgres

import (
	"context"
	"os"
	"testing"

	casesvc "opspilot-go/internal/case"
)

func TestCaseStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	want := casesvc.Case{
		ID:             "case-roundtrip-1",
		TenantID:       "tenant-1",
		Status:         casesvc.StatusOpen,
		Title:          "Review generated report",
		Summary:        "Operator wants a durable follow-up case.",
		SourceTaskID:   "task-source-1",
		SourceReportID: "report-source-1",
		CreatedBy:      "operator-1",
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO workflow_tasks (
    id, request_id, tenant_id, session_id, task_type, tool_name, tool_arguments,
    status, reason, error_reason, audit_ref, requires_approval, created_at, updated_at
) VALUES (
    'task-source-1', 'req-source-1', 'tenant-1', 'session-1', 'report_generation', '', '{}'::jsonb,
    'succeeded', 'workflow_required', '', '', false, NOW(), NOW()
)`); err != nil {
		t.Fatalf("insert workflow task error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO reports (
    id, tenant_id, source_task_id, report_type, status, title, summary, content_uri, metadata_json, created_by, created_at, ready_at
) VALUES (
    'report-source-1', 'tenant-1', 'task-source-1', 'workflow_summary', 'ready', 'Title', 'Summary', '', '{}'::jsonb, 'worker', NOW(), NOW()
)`); err != nil {
		t.Fatalf("insert report error = %v", err)
	}

	saved, err := store.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.ID != want.ID {
		t.Fatalf("Save().ID = %q, want %q", saved.ID, want.ID)
	}

	got, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != want.Title {
		t.Fatalf("Get().Title = %q, want %q", got.Title, want.Title)
	}
	if got.SourceTaskID != want.SourceTaskID {
		t.Fatalf("Get().SourceTaskID = %q, want %q", got.SourceTaskID, want.SourceTaskID)
	}
	if got.SourceReportID != want.SourceReportID {
		t.Fatalf("Get().SourceReportID = %q, want %q", got.SourceReportID, want.SourceReportID)
	}
}

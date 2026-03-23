package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

func TestReportStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	taskStore := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:        "task-report-store-1",
		RequestID: "req-report-store-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Status:    workflow.StatusSucceeded,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		CreatedAt: time.Unix(1700001100, 0).UTC(),
		UpdatedAt: time.Unix(1700001101, 0).UTC(),
	}
	if _, err := taskStore.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	store := NewReportStore(pool)
	readyAt := time.Unix(1700001102, 0).UTC()
	want := report.Report{
		ID:           "report-task-report-store-1",
		TenantID:     "tenant-1",
		SourceTaskID: task.ID,
		ReportType:   report.TypeWorkflowSummary,
		Status:       report.StatusReady,
		Title:        "Report for task-report-store-1",
		Summary:      "generated:task-report-store-1",
		MetadataJSON: json.RawMessage(`{"task_id":"task-report-store-1"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700001100, 0).UTC(),
		ReadyAt:      &readyAt,
	}

	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.SourceTaskID != want.SourceTaskID {
		t.Fatalf("SourceTaskID = %q, want %q", got.SourceTaskID, want.SourceTaskID)
	}
	if got.Summary != want.Summary {
		t.Fatalf("Summary = %q, want %q", got.Summary, want.Summary)
	}
}

package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/workflow"
)

func TestWorkflowTaskStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_tasks"); err != nil {
		t.Fatalf("TRUNCATE workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	want := workflow.Task{
		ID:               "task-test-roundtrip",
		RequestID:        "req-1",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000000, 0).UTC(),
		UpdatedAt:        time.Unix(1700000000, 0).UTC(),
	}

	got, err := store.SaveTask(ctx, want)
	if err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("SaveTask().ID = %q, want %q", got.ID, want.ID)
	}

	loaded, err := store.GetTask(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if loaded.ID != want.ID {
		t.Fatalf("GetTask().ID = %q, want %q", loaded.ID, want.ID)
	}
	if loaded.TenantID != want.TenantID {
		t.Fatalf("GetTask().TenantID = %q, want %q", loaded.TenantID, want.TenantID)
	}
	if loaded.Status != workflow.StatusQueued {
		t.Fatalf("GetTask().Status = %q, want %q", loaded.Status, workflow.StatusQueued)
	}
}

func TestWorkflowTaskStoreClaimAndUpdate(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_tasks"); err != nil {
		t.Fatalf("TRUNCATE workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	queuedTask := workflow.Task{
		ID:               "task-claim-1",
		RequestID:        "req-claim-1",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000001, 0).UTC(),
		UpdatedAt:        time.Unix(1700000001, 0).UTC(),
	}
	waitingApprovalTask := workflow.Task{
		ID:               "task-claim-2",
		RequestID:        "req-claim-2",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeApprovedToolExecution,
		Status:           workflow.StatusWaitingApproval,
		Reason:           workflow.PromotionReasonApprovalRequired,
		RequiresApproval: true,
		CreatedAt:        time.Unix(1700000002, 0).UTC(),
		UpdatedAt:        time.Unix(1700000002, 0).UTC(),
	}
	if _, err := store.SaveTask(ctx, queuedTask); err != nil {
		t.Fatalf("SaveTask(queued) error = %v", err)
	}
	if _, err := store.SaveTask(ctx, waitingApprovalTask); err != nil {
		t.Fatalf("SaveTask(waitingApproval) error = %v", err)
	}

	claimed, err := store.ClaimQueuedTasks(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedTasks() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(ClaimQueuedTasks()) = %d, want %d", len(claimed), 1)
	}
	if claimed[0].ID != queuedTask.ID {
		t.Fatalf("claimed[0].ID = %q, want %q", claimed[0].ID, queuedTask.ID)
	}
	if claimed[0].Status != workflow.StatusRunning {
		t.Fatalf("claimed[0].Status = %q, want %q", claimed[0].Status, workflow.StatusRunning)
	}

	claimed[0].Status = workflow.StatusSucceeded
	claimed[0].AuditRef = "worker:placeholder_report_generation"
	claimed[0].ErrorReason = ""
	claimed[0].UpdatedAt = time.Unix(1700000003, 0).UTC()

	updated, err := store.UpdateTask(ctx, claimed[0])
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Status != workflow.StatusSucceeded {
		t.Fatalf("updated.Status = %q, want %q", updated.Status, workflow.StatusSucceeded)
	}
	if updated.AuditRef != "worker:placeholder_report_generation" {
		t.Fatalf("updated.AuditRef = %q, want %q", updated.AuditRef, "worker:placeholder_report_generation")
	}
}

func applyMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	path := filepath.Join("..", "..", "..", "db", "migrations", "000002_workflow_tasks.sql")
	sql, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply migration error = %v", err)
	}
}

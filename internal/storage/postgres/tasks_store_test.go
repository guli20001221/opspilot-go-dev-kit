package postgres

import (
	"context"
	"encoding/json"
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	want := workflow.Task{
		ID:               "task-test-roundtrip",
		RequestID:        "req-1",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		ToolName:         "ticket_search",
		ToolArguments:    json.RawMessage(`{"query":"database incident"}`),
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
	if loaded.ToolName != want.ToolName {
		t.Fatalf("GetTask().ToolName = %q, want %q", loaded.ToolName, want.ToolName)
	}
	if string(loaded.ToolArguments) != string(want.ToolArguments) {
		t.Fatalf("GetTask().ToolArguments = %s, want %s", string(loaded.ToolArguments), string(want.ToolArguments))
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
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

func TestWorkflowTaskStoreAppendAndListTaskEvents(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:               "task-events-1",
		RequestID:        "req-events-1",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000010, 0).UTC(),
		UpdatedAt:        time.Unix(1700000010, 0).UTC(),
	}
	if _, err := store.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	if _, err := store.AppendTaskEvent(ctx, workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionCreated,
		Actor:     "api",
		Detail:    workflow.StatusQueued,
		CreatedAt: time.Unix(1700000010, 0).UTC(),
	}); err != nil {
		t.Fatalf("AppendTaskEvent(created) error = %v", err)
	}
	if _, err := store.AppendTaskEvent(ctx, workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionSucceeded,
		Actor:     "worker",
		Detail:    workflow.StatusSucceeded,
		CreatedAt: time.Unix(1700000011, 0).UTC(),
	}); err != nil {
		t.Fatalf("AppendTaskEvent(succeeded) error = %v", err)
	}

	events, err := store.ListTaskEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(events), 2)
	}
	if events[0].Action != workflow.AuditActionCreated {
		t.Fatalf("events[0].Action = %q, want %q", events[0].Action, workflow.AuditActionCreated)
	}
	if events[1].Action != workflow.AuditActionSucceeded {
		t.Fatalf("events[1].Action = %q, want %q", events[1].Action, workflow.AuditActionSucceeded)
	}
}

func TestWorkflowTaskStoreCreateTaskWithEventPersistsBoth(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:               "task-create-with-event",
		RequestID:        "req-create-with-event",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000020, 0).UTC(),
		UpdatedAt:        time.Unix(1700000020, 0).UTC(),
	}
	event := workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionCreated,
		Actor:     "api",
		Detail:    workflow.StatusQueued,
		CreatedAt: task.CreatedAt,
	}

	if _, err := store.CreateTaskWithEvent(ctx, task, event); err != nil {
		t.Fatalf("CreateTaskWithEvent() error = %v", err)
	}

	events, err := store.ListTaskEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want %d", len(events), 1)
	}
}

func TestWorkflowTaskStoreUpdateTaskWithEventPersistsBoth(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:               "task-update-with-event",
		RequestID:        "req-update-with-event",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000030, 0).UTC(),
		UpdatedAt:        time.Unix(1700000030, 0).UTC(),
	}
	if _, err := store.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	task.Status = workflow.StatusSucceeded
	task.UpdatedAt = time.Unix(1700000031, 0).UTC()
	event := workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionSucceeded,
		Actor:     "worker",
		Detail:    workflow.StatusSucceeded,
		CreatedAt: task.UpdatedAt,
	}

	if _, err := store.UpdateTaskWithEvent(ctx, task, event); err != nil {
		t.Fatalf("UpdateTaskWithEvent() error = %v", err)
	}

	loaded, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if loaded.Status != workflow.StatusSucceeded {
		t.Fatalf("loaded.Status = %q, want %q", loaded.Status, workflow.StatusSucceeded)
	}
	events, err := store.ListTaskEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want %d", len(events), 1)
	}
}

func TestWorkflowTaskStoreClaimQueuedTasksAppendsClaimedEvent(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE workflow_task_events, workflow_tasks RESTART IDENTITY"); err != nil {
		t.Fatalf("TRUNCATE workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:               "task-claim-with-event",
		RequestID:        "req-claim-with-event",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         workflow.TaskTypeReportGeneration,
		Status:           workflow.StatusQueued,
		Reason:           workflow.PromotionReasonWorkflowRequired,
		RequiresApproval: false,
		CreatedAt:        time.Unix(1700000040, 0).UTC(),
		UpdatedAt:        time.Unix(1700000040, 0).UTC(),
	}
	if _, err := store.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	claimed, err := store.ClaimQueuedTasks(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimQueuedTasks() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(claimed) = %d, want %d", len(claimed), 1)
	}

	events, err := store.ListTaskEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want %d", len(events), 1)
	}
	if events[0].Action != workflow.AuditActionClaimed {
		t.Fatalf("events[0].Action = %q, want %q", events[0].Action, workflow.AuditActionClaimed)
	}
}

func applyMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, name := range []string{
		"000002_workflow_tasks.sql",
		"000003_workflow_task_events.sql",
		"000004_workflow_task_payload.sql",
	} {
		path := filepath.Join("..", "..", "..", "db", "migrations", name)
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply migration %q error = %v", name, err)
		}
	}
}

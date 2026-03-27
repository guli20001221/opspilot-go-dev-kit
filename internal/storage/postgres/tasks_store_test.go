package postgres

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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
	if !jsonEqual(loaded.ToolArguments, want.ToolArguments) {
		t.Fatalf("GetTask().ToolArguments = %s, want %s", string(loaded.ToolArguments), string(want.ToolArguments))
	}
}

func TestWorkflowTaskStoreHandlesNullableToolColumns(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	const insert = `
INSERT INTO workflow_tasks (
    id,
    request_id,
    tenant_id,
    session_id,
    task_type,
    tool_name,
    tool_arguments,
    status,
    reason,
    error_reason,
    audit_ref,
    requires_approval,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, NULL, NULL, $6, $7, $8, $9, $10, $11, $12
)`
	createdAt := time.Unix(1700000005, 0).UTC()
	if _, err := pool.Exec(
		ctx,
		insert,
		"task-nullable-tools",
		"req-nullable-tools",
		"tenant-nullable-tools",
		"session-nullable-tools",
		workflow.TaskTypeReportGeneration,
		workflow.StatusSucceeded,
		workflow.PromotionReasonWorkflowRequired,
		"",
		"temporal:workflow:task-nullable-tools/run-1",
		false,
		createdAt,
		createdAt,
	); err != nil {
		t.Fatalf("insert nullable workflow task error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	got, err := store.GetTask(ctx, "task-nullable-tools")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.ToolName != "" {
		t.Fatalf("GetTask().ToolName = %q, want empty string", got.ToolName)
	}
	if got.ToolArguments != nil {
		t.Fatalf("GetTask().ToolArguments = %v, want nil", got.ToolArguments)
	}

	page, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID: "tenant-nullable-tools",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(page.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(page.Tasks), 1)
	}
	if page.Tasks[0].ToolName != "" {
		t.Fatalf("ListTasks().Tasks[0].ToolName = %q, want empty string", page.Tasks[0].ToolName)
	}
	if page.Tasks[0].ToolArguments != nil {
		t.Fatalf("ListTasks().Tasks[0].ToolArguments = %v, want nil", page.Tasks[0].ToolArguments)
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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

func TestWorkflowTaskStoreListTasksSupportsFiltersAndLimit(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	tasks := []workflow.Task{
		{
			ID:        "task-list-1",
			RequestID: "req-list-1",
			TenantID:  "tenant-1",
			SessionID: "session-1",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000100, 0).UTC(),
			UpdatedAt: time.Unix(1700000101, 0).UTC(),
		},
		{
			ID:               "task-list-2",
			RequestID:        "req-list-2",
			TenantID:         "tenant-1",
			SessionID:        "session-2",
			TaskType:         workflow.TaskTypeApprovedToolExecution,
			Status:           workflow.StatusFailed,
			Reason:           workflow.PromotionReasonApprovalRequired,
			RequiresApproval: true,
			CreatedAt:        time.Unix(1700000102, 0).UTC(),
			UpdatedAt:        time.Unix(1700000103, 0).UTC(),
		},
		{
			ID:        "task-list-3",
			RequestID: "req-list-3",
			TenantID:  "tenant-2",
			SessionID: "session-3",
			TaskType:  workflow.TaskTypeApprovedToolExecution,
			Status:    workflow.StatusFailed,
			Reason:    workflow.PromotionReasonApprovalRequired,
			CreatedAt: time.Unix(1700000104, 0).UTC(),
			UpdatedAt: time.Unix(1700000105, 0).UTC(),
		},
	}
	for _, task := range tasks {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	got, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID: "tenant-1",
		Status:   workflow.StatusFailed,
		TaskType: workflow.TaskTypeApprovedToolExecution,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].ID != "task-list-2" {
		t.Fatalf("ListTasks().Tasks[0].ID = %q, want %q", got.Tasks[0].ID, "task-list-2")
	}
	if got.HasMore {
		t.Fatal("HasMore = true, want false")
	}
}

func TestWorkflowTaskStoreListTasksSupportsReasonAndApprovalFilters(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	tasks := []workflow.Task{
		{
			ID:        "task-filter-1",
			RequestID: "req-filter-1",
			TenantID:  "tenant-filter",
			SessionID: "session-filter-1",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000300, 0).UTC(),
			UpdatedAt: time.Unix(1700000301, 0).UTC(),
		},
		{
			ID:               "task-filter-2",
			RequestID:        "req-filter-2",
			TenantID:         "tenant-filter",
			SessionID:        "session-filter-2",
			TaskType:         workflow.TaskTypeApprovedToolExecution,
			Status:           workflow.StatusWaitingApproval,
			Reason:           workflow.PromotionReasonApprovalRequired,
			RequiresApproval: true,
			CreatedAt:        time.Unix(1700000302, 0).UTC(),
			UpdatedAt:        time.Unix(1700000303, 0).UTC(),
		},
	}
	for _, task := range tasks {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	got, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID:         "tenant-filter",
		Reason:           workflow.PromotionReasonApprovalRequired,
		RequiresApproval: boolPtr(true),
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].ID != "task-filter-2" {
		t.Fatalf("ListTasks().Tasks[0].ID = %q, want %q", got.Tasks[0].ID, "task-filter-2")
	}
}

func TestWorkflowTaskStoreListTasksSupportsCreatedAtWindowFilters(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	firstCreatedAt := time.Unix(1700000400, 0).UTC()
	secondCreatedAt := time.Unix(1700000401, 0).UTC()
	thirdCreatedAt := time.Unix(1700000402, 0).UTC()
	tasks := []workflow.Task{
		{
			ID:        "task-time-1",
			RequestID: "req-time-1",
			TenantID:  "tenant-time",
			SessionID: "session-time-1",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: firstCreatedAt,
			UpdatedAt: firstCreatedAt,
		},
		{
			ID:        "task-time-2",
			RequestID: "req-time-2",
			TenantID:  "tenant-time",
			SessionID: "session-time-2",
			TaskType:  workflow.TaskTypeApprovedToolExecution,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonApprovalRequired,
			CreatedAt: secondCreatedAt,
			UpdatedAt: secondCreatedAt,
		},
		{
			ID:        "task-time-3",
			RequestID: "req-time-3",
			TenantID:  "tenant-time",
			SessionID: "session-time-3",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: thirdCreatedAt,
			UpdatedAt: thirdCreatedAt,
		},
	}
	for _, task := range tasks {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	got, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID:      "tenant-time",
		CreatedAfter:  timePtr(firstCreatedAt),
		CreatedBefore: timePtr(thirdCreatedAt),
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].ID != "task-time-2" {
		t.Fatalf("ListTasks().Tasks[0].ID = %q, want %q", got.Tasks[0].ID, "task-time-2")
	}
}

func TestWorkflowTaskStoreListTasksSupportsUpdatedAtWindowFilters(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	firstUpdatedAt := time.Unix(1700000500, 0).UTC()
	secondUpdatedAt := time.Unix(1700000501, 0).UTC()
	thirdUpdatedAt := time.Unix(1700000502, 0).UTC()
	tasks := []workflow.Task{
		{
			ID:        "task-updated-1",
			RequestID: "req-updated-1",
			TenantID:  "tenant-updated",
			SessionID: "session-updated-1",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000490, 0).UTC(),
			UpdatedAt: firstUpdatedAt,
		},
		{
			ID:        "task-updated-2",
			RequestID: "req-updated-2",
			TenantID:  "tenant-updated",
			SessionID: "session-updated-2",
			TaskType:  workflow.TaskTypeApprovedToolExecution,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonApprovalRequired,
			CreatedAt: time.Unix(1700000491, 0).UTC(),
			UpdatedAt: secondUpdatedAt,
		},
		{
			ID:        "task-updated-3",
			RequestID: "req-updated-3",
			TenantID:  "tenant-updated",
			SessionID: "session-updated-3",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusQueued,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000492, 0).UTC(),
			UpdatedAt: thirdUpdatedAt,
		},
	}
	for _, task := range tasks {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	got, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID:      "tenant-updated",
		UpdatedAfter:  timePtr(firstUpdatedAt),
		UpdatedBefore: timePtr(thirdUpdatedAt),
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].ID != "task-updated-2" {
		t.Fatalf("ListTasks().Tasks[0].ID = %q, want %q", got.Tasks[0].ID, "task-updated-2")
	}
}

func TestWorkflowTaskStoreListTasksSupportsOffsetPagination(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	tasks := []workflow.Task{
		{
			ID:        "task-page-1",
			RequestID: "req-page-1",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000200, 0).UTC(),
			UpdatedAt: time.Unix(1700000201, 0).UTC(),
		},
		{
			ID:        "task-page-2",
			RequestID: "req-page-2",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000202, 0).UTC(),
			UpdatedAt: time.Unix(1700000203, 0).UTC(),
		},
		{
			ID:        "task-page-3",
			RequestID: "req-page-3",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700000204, 0).UTC(),
			UpdatedAt: time.Unix(1700000205, 0).UTC(),
		},
	}
	for _, task := range tasks {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	got, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID: "tenant-page",
		Limit:    1,
		Offset:   1,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(ListTasks().Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].ID != "task-page-2" {
		t.Fatalf("ListTasks().Tasks[0].ID = %q, want %q", got.Tasks[0].ID, "task-page-2")
	}
	if !got.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if got.NextOffset != 2 {
		t.Fatalf("NextOffset = %d, want %d", got.NextOffset, 2)
	}
}

func TestWorkflowTaskStoreListTasksUsesStableIDTieBreakForPagination(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewWorkflowTaskStore(pool)
	ts := time.Unix(1700000600, 0).UTC()
	for _, task := range []workflow.Task{
		{
			ID:        "task-page-1",
			RequestID: "req-page-1",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
		{
			ID:        "task-page-2",
			RequestID: "req-page-2",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
		{
			ID:        "task-page-3",
			RequestID: "req-page-3",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
	} {
		if _, err := store.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	firstPage, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID: "tenant-page",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListTasks(firstPage) error = %v", err)
	}
	secondPage, err := store.ListTasks(ctx, workflow.TaskListFilter{
		TenantID: "tenant-page",
		Limit:    1,
		Offset:   1,
	})
	if err != nil {
		t.Fatalf("ListTasks(secondPage) error = %v", err)
	}

	if len(firstPage.Tasks) != 1 {
		t.Fatalf("len(firstPage.Tasks) = %d, want %d", len(firstPage.Tasks), 1)
	}
	if len(secondPage.Tasks) != 1 {
		t.Fatalf("len(secondPage.Tasks) = %d, want %d", len(secondPage.Tasks), 1)
	}
	if firstPage.Tasks[0].ID != "task-page-3" {
		t.Fatalf("firstPage.Tasks[0].ID = %q, want %q", firstPage.Tasks[0].ID, "task-page-3")
	}
	if secondPage.Tasks[0].ID != "task-page-2" {
		t.Fatalf("secondPage.Tasks[0].ID = %q, want %q", secondPage.Tasks[0].ID, "task-page-2")
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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
	if _, err := pool.Exec(ctx, "TRUNCATE reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE reports, workflow_task_events, workflow_tasks error = %v", err)
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
		"000005_reports.sql",
		"000006_cases.sql",
		"000007_case_close_fields.sql",
		"000008_case_assignment_fields.sql",
		"000009_case_notes.sql",
		"000010_versions.sql",
		"000011_version_refs.sql",
		"000012_eval_cases.sql",
		"000013_eval_datasets.sql",
		"000014_eval_dataset_publish_fields.sql",
		"000015_eval_runs.sql",
		"000016_eval_run_events.sql",
		"000017_eval_run_items.sql",
		"000018_eval_run_item_results.sql",
		"000019_eval_run_item_judge_fields.sql",
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

func jsonEqual(left json.RawMessage, right json.RawMessage) bool {
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}

	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}

	return reflect.DeepEqual(leftValue, rightValue)
}

func boolPtr(v bool) *bool {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}

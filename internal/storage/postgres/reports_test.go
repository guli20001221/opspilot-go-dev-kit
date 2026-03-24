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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
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

func TestReportStoreFinalizeSucceededTaskWithReport(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	taskStore := NewWorkflowTaskStore(pool)
	task := workflow.Task{
		ID:        "task-report-finalize-1",
		RequestID: "req-report-finalize-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Status:    workflow.StatusRunning,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		CreatedAt: time.Unix(1700001300, 0).UTC(),
		UpdatedAt: time.Unix(1700001301, 0).UTC(),
	}
	if _, err := taskStore.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	readyAt := time.Unix(1700001302, 0).UTC()
	finalTask := task
	finalTask.Status = workflow.StatusSucceeded
	finalTask.VersionID = "version-skeleton-2026-03-24"
	finalTask.AuditRef = "temporal:workflow:task-report-finalize-1/run-1"
	finalTask.UpdatedAt = readyAt
	reportItem := report.Report{
		ID:           report.ReportIDFromTaskID(task.ID),
		TenantID:     task.TenantID,
		SourceTaskID: task.ID,
		VersionID:    finalTask.VersionID,
		ReportType:   report.TypeWorkflowSummary,
		Status:       report.StatusReady,
		Title:        "Report for task-report-finalize-1",
		Summary:      "generated:task-report-finalize-1",
		MetadataJSON: json.RawMessage(`{"audit_ref":"temporal:workflow:task-report-finalize-1/run-1"}`),
		CreatedBy:    "worker",
		CreatedAt:    task.CreatedAt,
		ReadyAt:      &readyAt,
	}

	store := NewReportStore(pool)
	saved, updated, err := store.FinalizeSucceededTaskWithReport(ctx, finalTask, workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionSucceeded,
		Actor:     "worker",
		Detail:    "generated:task-report-finalize-1",
		CreatedAt: readyAt,
	}, reportItem)
	if err != nil {
		t.Fatalf("FinalizeSucceededTaskWithReport() error = %v", err)
	}

	if updated.Status != workflow.StatusSucceeded {
		t.Fatalf("updated.Status = %q, want %q", updated.Status, workflow.StatusSucceeded)
	}
	if updated.AuditRef != "temporal:workflow:task-report-finalize-1/run-1" {
		t.Fatalf("updated.AuditRef = %q, want final temporal ref", updated.AuditRef)
	}
	if updated.VersionID != finalTask.VersionID {
		t.Fatalf("updated.VersionID = %q, want %q", updated.VersionID, finalTask.VersionID)
	}
	if saved.ID != report.ReportIDFromTaskID(task.ID) {
		t.Fatalf("saved.ID = %q, want %q", saved.ID, report.ReportIDFromTaskID(task.ID))
	}

	events, err := taskStore.ListTaskEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Action != workflow.AuditActionSucceeded {
		t.Fatalf("events = %#v, want single succeeded event", events)
	}

	got, err := store.Get(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(got.MetadataJSON, &metadata); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	if metadata["audit_ref"] != "temporal:workflow:task-report-finalize-1/run-1" {
		t.Fatalf("metadata audit_ref = %v, want final temporal ref", metadata["audit_ref"])
	}
	if got.VersionID != finalTask.VersionID {
		t.Fatalf("got.VersionID = %q, want %q", got.VersionID, finalTask.VersionID)
	}
}

func TestReportStoreListAppliesFiltersAndPagination(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewReportStore(pool)
	readyOne := time.Unix(1700002100, 0).UTC()
	readyTwo := time.Unix(1700002200, 0).UTC()
	readyThree := time.Unix(1700002300, 0).UTC()
	taskStore := NewWorkflowTaskStore(pool)
	for _, task := range []workflow.Task{
		{
			ID:        "task-a",
			RequestID: "req-task-a",
			TenantID:  "tenant-a",
			SessionID: "session-a",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700002050, 0).UTC(),
			UpdatedAt: time.Unix(1700002051, 0).UTC(),
		},
		{
			ID:        "task-b",
			RequestID: "req-task-b",
			TenantID:  "tenant-a",
			SessionID: "session-b",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700002052, 0).UTC(),
			UpdatedAt: time.Unix(1700002053, 0).UTC(),
		},
		{
			ID:        "task-c",
			RequestID: "req-task-c",
			TenantID:  "tenant-b",
			SessionID: "session-c",
			TaskType:  workflow.TaskTypeReportGeneration,
			Status:    workflow.StatusSucceeded,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: time.Unix(1700002054, 0).UTC(),
			UpdatedAt: time.Unix(1700002055, 0).UTC(),
		},
	} {
		if _, err := taskStore.SaveTask(ctx, task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}
	fixtures := []report.Report{
		{
			ID:           "report-list-a",
			TenantID:     "tenant-a",
			SourceTaskID: "task-a",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report A",
			Summary:      "A",
			MetadataJSON: json.RawMessage(`{"task_id":"task-a"}`),
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002000, 0).UTC(),
			ReadyAt:      &readyOne,
		},
		{
			ID:           "report-list-b",
			TenantID:     "tenant-a",
			SourceTaskID: "task-b",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report B",
			Summary:      "B",
			MetadataJSON: json.RawMessage(`{"task_id":"task-b"}`),
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002001, 0).UTC(),
			ReadyAt:      &readyTwo,
		},
		{
			ID:           "report-list-c",
			TenantID:     "tenant-b",
			SourceTaskID: "task-c",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report C",
			Summary:      "C",
			MetadataJSON: json.RawMessage(`{"task_id":"task-c"}`),
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002002, 0).UTC(),
			ReadyAt:      &readyThree,
		},
	}

	for _, item := range fixtures {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(ctx, report.ListFilter{
		TenantID:   "tenant-a",
		Status:     report.StatusReady,
		ReportType: report.TypeWorkflowSummary,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Reports) != 1 || page.Reports[0].ID != "report-list-b" {
		t.Fatalf("page.Reports = %#v, want report-list-b first", page.Reports)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}

	nextPage, err := store.List(ctx, report.ListFilter{
		TenantID:   "tenant-a",
		Status:     report.StatusReady,
		ReportType: report.TypeWorkflowSummary,
		Limit:      1,
		Offset:     1,
	})
	if err != nil {
		t.Fatalf("List(offset) error = %v", err)
	}
	if len(nextPage.Reports) != 1 || nextPage.Reports[0].ID != "report-list-a" {
		t.Fatalf("nextPage.Reports = %#v, want report-list-a", nextPage.Reports)
	}
	if nextPage.HasMore {
		t.Fatalf("HasMore = true, want false on final page")
	}
}

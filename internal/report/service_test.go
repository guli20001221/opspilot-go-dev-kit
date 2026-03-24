package report

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"opspilot-go/internal/workflow"
)

func TestServiceRecordGeneratedReport(t *testing.T) {
	svc := NewService()
	task := workflow.Task{
		ID:        "task-report-1",
		RequestID: "req-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		AuditRef:  "temporal:workflow:task-report-1/run-1",
		CreatedAt: time.Unix(1700001000, 0).UTC(),
		UpdatedAt: time.Unix(1700001001, 0).UTC(),
	}

	reportID, err := svc.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated:task-report-1",
	})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}
	if reportID != "report-task-report-1" {
		t.Fatalf("reportID = %q, want %q", reportID, "report-task-report-1")
	}

	got, err := svc.GetReport(context.Background(), reportID)
	if err != nil {
		t.Fatalf("GetReport() error = %v", err)
	}
	if got.SourceTaskID != task.ID {
		t.Fatalf("SourceTaskID = %q, want %q", got.SourceTaskID, task.ID)
	}
	if got.Status != StatusReady {
		t.Fatalf("Status = %q, want %q", got.Status, StatusReady)
	}
	if got.Summary != "generated:task-report-1" {
		t.Fatalf("Summary = %q, want execution detail", got.Summary)
	}

	var metadata map[string]any
	if err := json.Unmarshal(got.MetadataJSON, &metadata); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	if metadata["task_id"] != task.ID {
		t.Fatalf("metadata task_id = %v, want %q", metadata["task_id"], task.ID)
	}
}

func TestServiceSupportsAtomicFinalizationDependsOnStore(t *testing.T) {
	if NewService().SupportsAtomicFinalization() {
		t.Fatal("SupportsAtomicFinalization() = true, want false for default memory store")
	}
}

func TestServiceListReportsAppliesFiltersAndPagination(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	readyOne := time.Unix(1700002003, 0).UTC()
	readyTwo := time.Unix(1700002005, 0).UTC()
	readyThree := time.Unix(1700002004, 0).UTC()
	items := []Report{
		{
			ID:           "report-a",
			TenantID:     "tenant-a",
			SourceTaskID: "task-a",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "A",
			Summary:      "A",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002000, 0).UTC(),
			ReadyAt:      &readyOne,
		},
		{
			ID:           "report-b",
			TenantID:     "tenant-a",
			SourceTaskID: "task-b",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "B",
			Summary:      "B",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002001, 0).UTC(),
			ReadyAt:      &readyTwo,
		},
		{
			ID:           "report-c",
			TenantID:     "tenant-b",
			SourceTaskID: "task-c",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "C",
			Summary:      "C",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002002, 0).UTC(),
			ReadyAt:      &readyThree,
		},
	}

	for _, item := range items {
		if _, err := svc.store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := svc.ListReports(ctx, ListFilter{
		TenantID:   "tenant-a",
		Status:     StatusReady,
		ReportType: TypeWorkflowSummary,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if len(page.Reports) != 1 {
		t.Fatalf("len(Reports) = %d, want 1", len(page.Reports))
	}
	if page.Reports[0].ID != "report-b" {
		t.Fatalf("Reports[0].ID = %q, want report-b", page.Reports[0].ID)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}

	nextPage, err := svc.ListReports(ctx, ListFilter{
		TenantID:   "tenant-a",
		Status:     StatusReady,
		ReportType: TypeWorkflowSummary,
		Limit:      1,
		Offset:     1,
	})
	if err != nil {
		t.Fatalf("ListReports(offset) error = %v", err)
	}
	if len(nextPage.Reports) != 1 || nextPage.Reports[0].ID != "report-a" {
		t.Fatalf("next page = %#v, want report-a", nextPage.Reports)
	}
	if nextPage.HasMore {
		t.Fatalf("HasMore = true, want false on final page")
	}
}

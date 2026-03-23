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

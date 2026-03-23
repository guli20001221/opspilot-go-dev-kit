package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

func TestGetReportReturnsStoredReport(t *testing.T) {
	reportService := report.NewService()
	task := workflow.Task{
		ID:        "task-report-http-1",
		RequestID: "req-report-http-1",
		TenantID:  "tenant-report-http",
		SessionID: "session-report-http",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		AuditRef:  "temporal:workflow:task-report-http-1/run-1",
		CreatedAt: time.Unix(1700001200, 0).UTC(),
		UpdatedAt: time.Unix(1700001201, 0).UTC(),
	}
	if _, err := reportService.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated:task-report-http-1",
	}); err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports/report-task-report-http-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body reportResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.ReportID != "report-task-report-http-1" {
		t.Fatalf("ReportID = %q, want %q", body.ReportID, "report-task-report-http-1")
	}
	if body.SourceTaskID != "task-report-http-1" {
		t.Fatalf("SourceTaskID = %q, want %q", body.SourceTaskID, "task-report-http-1")
	}
	if body.Status != report.StatusReady {
		t.Fatalf("Status = %q, want %q", body.Status, report.StatusReady)
	}
	if body.ReadyAt == "" {
		t.Fatal("ReadyAt is empty")
	}
	if string(body.Metadata) == "" {
		t.Fatal("Metadata is empty")
	}
}

func TestGetReportReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: report.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports/report-missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

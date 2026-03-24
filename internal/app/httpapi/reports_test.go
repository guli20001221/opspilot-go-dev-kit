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

func TestListReportsReturnsFilteredPage(t *testing.T) {
	reportService := report.NewService()
	ctx := context.Background()
	fixtures := []report.Report{
		{
			ID:           "report-list-http-a",
			TenantID:     "tenant-report-list",
			SourceTaskID: "task-a",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report A",
			Summary:      "A",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700003000, 0).UTC(),
			ReadyAt:      ptrTime(time.Unix(1700003002, 0).UTC()),
		},
		{
			ID:           "report-list-http-b",
			TenantID:     "tenant-report-list",
			SourceTaskID: "task-b",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report B",
			Summary:      "B",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700003001, 0).UTC(),
			ReadyAt:      ptrTime(time.Unix(1700003003, 0).UTC()),
		},
		{
			ID:           "report-list-http-c",
			TenantID:     "tenant-other",
			SourceTaskID: "task-c",
			ReportType:   report.TypeWorkflowSummary,
			Status:       report.StatusReady,
			Title:        "Report C",
			Summary:      "C",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700003004, 0).UTC(),
			ReadyAt:      ptrTime(time.Unix(1700003005, 0).UTC()),
		},
	}
	for _, item := range fixtures {
		if _, err := reportService.RecordGeneratedReport(ctx, workflow.Task{
			ID:        item.SourceTaskID,
			RequestID: "req-" + item.SourceTaskID,
			TenantID:  item.TenantID,
			SessionID: "session-" + item.SourceTaskID,
			TaskType:  workflow.TaskTypeReportGeneration,
			Reason:    workflow.PromotionReasonWorkflowRequired,
			CreatedAt: item.CreatedAt,
			UpdatedAt: *item.ReadyAt,
		}, workflow.ExecutionResult{Detail: item.Summary}); err != nil {
			t.Fatalf("RecordGeneratedReport(%s) error = %v", item.ID, err)
		}
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: reportService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports?tenant_id=tenant-report-list&status=ready&report_type=workflow_summary&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body listReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(body.Reports) != 1 || body.Reports[0].ReportID != "report-task-b" {
		t.Fatalf("Reports = %#v, want report-task-b first", body.Reports)
	}
	if !body.HasMore || body.NextOffset == nil || *body.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", body)
	}
}

func TestListReportsRejectsInvalidOffset(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: report.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports?offset=-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestListReportsRejectsInvalidStatus(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: report.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports?status=running")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestListReportsRejectsInvalidReportType(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: report.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/reports?report_type=custom")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

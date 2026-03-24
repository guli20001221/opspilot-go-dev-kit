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

func TestReportCompareReturnsDerivedSummary(t *testing.T) {
	reportService := report.NewService()
	ctx := context.Background()
	leftReady := time.Unix(1700005000, 0).UTC()
	rightReady := time.Unix(1700005015, 0).UTC()
	left := report.Report{
		ID:           "report-http-compare-left",
		TenantID:     "tenant-http-compare",
		SourceTaskID: "task-http-left",
		ReportType:   report.TypeWorkflowSummary,
		Status:       report.StatusReady,
		Title:        "Left",
		Summary:      "left summary",
		ContentURI:   "s3://left",
		MetadataJSON: json.RawMessage(`{"version":"left"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700004990, 0).UTC(),
		ReadyAt:      &leftReady,
	}
	right := report.Report{
		ID:           "report-http-compare-right",
		TenantID:     "tenant-http-compare",
		SourceTaskID: "task-http-right",
		ReportType:   report.TypeWorkflowSummary,
		Status:       report.StatusReady,
		Title:        "Right",
		Summary:      "right summary",
		ContentURI:   "s3://right",
		MetadataJSON: json.RawMessage(`{"version":"right"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700004995, 0).UTC(),
		ReadyAt:      &rightReady,
	}
	for _, item := range []report.Report{left, right} {
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

	resp, err := http.Get(server.URL + "/api/v1/report-compare?left_report_id=report-task-http-left&right_report_id=report-task-http-right")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body reportComparisonResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.Left.ReportID != "report-task-http-left" || body.Right.ReportID != "report-task-http-right" {
		t.Fatalf("comparison reports = %#v", body)
	}
	if !body.Summary.SameTenant || !body.Summary.SourceTaskChanged || !body.Summary.SummaryChanged {
		t.Fatalf("summary = %#v, want same tenant and changed source/summary", body.Summary)
	}
	if body.Summary.ReadyAtDeltaSecond != 15 {
		t.Fatalf("ReadyAtDeltaSecond = %d, want 15", body.Summary.ReadyAtDeltaSecond)
	}
}

func TestReportCompareRejectsMissingRightReportID(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Reports: report.NewService()}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/report-compare?left_report_id=report-a")
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

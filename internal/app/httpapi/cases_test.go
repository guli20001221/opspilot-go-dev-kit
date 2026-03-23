package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	casesvc "opspilot-go/internal/case"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

func TestCreateAndGetCaseEndpoint(t *testing.T) {
	workflowService := workflow.NewService()
	reportService := report.NewService()
	caseService := casesvc.NewService()

	task, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-case-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	task.Status = workflow.StatusSucceeded
	task.AuditRef = "temporal:workflow:task-1/run-1"
	if _, err := workflowService.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	reportID, err := reportService.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated report",
	})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Workflows: workflowService,
		Reports:   reportService,
		Cases:     caseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","title":"Investigate report","summary":"Review the generated report","source_task_id":"` + task.ID + `","source_report_id":"` + reportID + `","created_by":"operator-1"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if created.CaseID == "" {
		t.Fatal("case_id is empty")
	}
	if created.Status != casesvc.StatusOpen {
		t.Fatalf("Status = %q, want %q", created.Status, casesvc.StatusOpen)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode(get) error = %v", err)
	}
	if got.SourceReportID != reportID {
		t.Fatalf("SourceReportID = %q, want %q", got.SourceReportID, reportID)
	}
}

func TestCreateCaseRejectsCrossTenantSources(t *testing.T) {
	workflowService := workflow.NewService()
	reportService := report.NewService()
	caseService := casesvc.NewService()

	task, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-case-cross-tenant",
		TenantID:  "tenant-a",
		SessionID: "session-a",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Workflows: workflowService,
		Reports:   reportService,
		Cases:     caseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-b","title":"Cross tenant","source_task_id":"` + task.ID + `"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_case_source" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_case_source")
	}
}

func TestCreateCaseRejectsMismatchedReportTaskSource(t *testing.T) {
	workflowService := workflow.NewService()
	reportService := report.NewService()
	caseService := casesvc.NewService()

	task1, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-case-task-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(task1) error = %v", err)
	}
	task1.Status = workflow.StatusSucceeded
	if _, err := workflowService.UpdateTask(context.Background(), task1); err != nil {
		t.Fatalf("UpdateTask(task1) error = %v", err)
	}
	reportID, err := reportService.RecordGeneratedReport(context.Background(), task1, workflow.ExecutionResult{Detail: "generated report"})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}

	task2, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-case-task-2",
		TenantID:  "tenant-1",
		SessionID: "session-2",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(task2) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Workflows: workflowService,
		Reports:   reportService,
		Cases:     caseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","title":"Mismatch","source_task_id":"` + task2.ID + `","source_report_id":"` + reportID + `"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_case_source" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_case_source")
	}
}

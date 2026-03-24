package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cases "opspilot-go/internal/case"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

func TestTraceDrilldownReturnsTaskSubject(t *testing.T) {
	workflowService := workflow.NewService()
	ctx := context.Background()
	now := time.Unix(1700007000, 0).UTC()
	task, err := workflowService.Promote(ctx, workflow.PromoteRequest{
		RequestID: "req-trace-task",
		TenantID:  "tenant-trace",
		SessionID: "session-trace",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	task.Status = workflow.StatusSucceeded
	task.AuditRef = "temporal:workflow:" + task.ID + "/run-1"
	task.UpdatedAt = now
	if _, err := workflowService.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/trace-drilldown?task_id=" + task.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body traceDrilldownResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.Subject.Kind != "task" || body.Subject.ID != task.ID {
		t.Fatalf("Subject = %#v, want task subject", body.Subject)
	}
	if body.RequestID != "req-trace-task" || body.SessionID != "session-trace" {
		t.Fatalf("request/session = %#v", body)
	}
	if body.Temporal == nil || body.Temporal.WorkflowID != task.ID || body.Temporal.RunID != "run-1" {
		t.Fatalf("Temporal = %#v, want %s/run-1", body.Temporal, task.ID)
	}
}

func TestTraceDrilldownReturnsCaseLineage(t *testing.T) {
	workflowService := workflow.NewService()
	reportService := report.NewService()
	caseService := cases.NewService()
	ctx := context.Background()

	task, err := workflowService.Promote(ctx, workflow.PromoteRequest{
		RequestID: "req-trace-case",
		TenantID:  "tenant-trace-case",
		SessionID: "session-trace-case",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	task.Status = workflow.StatusSucceeded
	task.AuditRef = "temporal:workflow:" + task.ID + "/run-case"
	task.UpdatedAt = time.Unix(1700007100, 0).UTC()
	if _, err := workflowService.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	reportID, err := reportService.RecordGeneratedReport(ctx, task, workflow.ExecutionResult{Detail: "generated:" + task.ID})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}
	createdCase, err := caseService.CreateCase(ctx, cases.CreateInput{
		TenantID:       task.TenantID,
		Title:          "Trace case",
		SourceTaskID:   task.ID,
		SourceReportID: reportID,
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Workflows: workflowService,
		Reports:   reportService,
		Cases:     caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/trace-drilldown?case_id=" + createdCase.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body traceDrilldownResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.Subject.Kind != "case" || body.Lineage.CaseID != createdCase.ID {
		t.Fatalf("Subject/lineage = %#v", body)
	}
	if body.Lineage.TaskID != task.ID || body.Lineage.ReportID != reportID {
		t.Fatalf("Lineage = %#v, want task/report lineage", body.Lineage)
	}
	if body.CaseStatus != cases.StatusOpen || body.ReportStatus != report.StatusReady || body.TaskStatus != workflow.StatusSucceeded {
		t.Fatalf("statuses = %#v", body)
	}
}

func TestTraceDrilldownRejectsAmbiguousQuery(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/trace-drilldown?task_id=task-1&report_id=report-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

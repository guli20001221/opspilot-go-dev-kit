package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
	"opspilot-go/internal/report"
	"opspilot-go/internal/version"
	"opspilot-go/internal/workflow"
)

func TestCreateAndGetEvalCaseEndpoint(t *testing.T) {
	workflowService := workflow.NewService()
	reportService := report.NewService()
	caseService := casesvc.NewService()
	versionService := version.NewService()

	task, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-eval-case-1",
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
	task.VersionID = version.DefaultVersionID
	if _, err := workflowService.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	reportID, err := reportService.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated report for eval promotion",
	})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}
	createdCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:       "tenant-1",
		Title:          "Regression gap",
		Summary:        "Promote this case into durable eval coverage.",
		SourceTaskID:   task.ID,
		SourceReportID: reportID,
		CreatedBy:      "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Workflows: workflowService,
		Reports:   reportService,
		Cases:     caseService,
		Versions:  versionService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-cases", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-1","source_case_id":"`+createdCase.ID+`","operator_note":"capture for regression","created_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created evalCaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if created.EvalCaseID == "" {
		t.Fatal("eval_case_id is empty")
	}
	if created.SourceCaseID != createdCase.ID {
		t.Fatalf("SourceCaseID = %q, want %q", created.SourceCaseID, createdCase.ID)
	}
	if created.SourceReportID != reportID {
		t.Fatalf("SourceReportID = %q, want %q", created.SourceReportID, reportID)
	}
	if created.VersionID != version.DefaultVersionID {
		t.Fatalf("VersionID = %q, want %q", created.VersionID, version.DefaultVersionID)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-cases/" + created.EvalCaseID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusBadRequest)
	}

	getResp, err = http.Get(server.URL + "/api/v1/eval-cases/" + created.EvalCaseID + "?tenant_id=tenant-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var got evalCaseResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode(get) error = %v", err)
	}
	if got.EvalCaseID != created.EvalCaseID {
		t.Fatalf("EvalCaseID = %q, want %q", got.EvalCaseID, created.EvalCaseID)
	}
}

func TestCreateEvalCaseEndpointIsIdempotentBySourceCase(t *testing.T) {
	caseService := casesvc.NewService()
	createdCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Promote once",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	body := `{"tenant_id":"tenant-1","source_case_id":"` + createdCase.ID + `"}`
	firstResp, err := http.Post(server.URL+"/api/v1/eval-cases", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Post(first) error = %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("first StatusCode = %d, want %d", firstResp.StatusCode, http.StatusCreated)
	}
	var first evalCaseResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&first); err != nil {
		t.Fatalf("Decode(first) error = %v", err)
	}

	secondResp, err := http.Post(server.URL+"/api/v1/eval-cases", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Post(second) error = %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("second StatusCode = %d, want %d", secondResp.StatusCode, http.StatusOK)
	}
	var second evalCaseResponse
	if err := json.NewDecoder(secondResp.Body).Decode(&second); err != nil {
		t.Fatalf("Decode(second) error = %v", err)
	}
	if second.EvalCaseID != first.EvalCaseID {
		t.Fatalf("second.EvalCaseID = %q, want %q", second.EvalCaseID, first.EvalCaseID)
	}
}

func TestCreateEvalCaseEndpointRejectsCrossTenantSource(t *testing.T) {
	caseService := casesvc.NewService()
	createdCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-a",
		Title:    "Cross tenant",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-cases", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-b","source_case_id":"`+createdCase.ID+`"}`))
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
	if got.Code != "invalid_eval_case_source" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_eval_case_source")
	}
}

func TestGetEvalCaseEndpointFailsClosedForWrongTenant(t *testing.T) {
	caseService := casesvc.NewService()
	createdCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-a",
		Title:    "Tenant-safe eval case",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/eval-cases", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-a","source_case_id":"`+createdCase.ID+`"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	var created evalCaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	getResp, err := http.Get(server.URL + "/api/v1/eval-cases/" + created.EvalCaseID + "?tenant_id=tenant-b")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
}

func TestListEvalCasesEndpointSupportsFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := evalsvc.NewService(caseService, nil)

	caseA, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:       "tenant-list",
		Title:          "Eval A",
		Summary:        "A",
		SourceTaskID:   "task-a",
		SourceReportID: "report-a",
	})
	if err != nil {
		t.Fatalf("CreateCase(caseA) error = %v", err)
	}
	first, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-list",
		SourceCaseID: caseA.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)

	caseB, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:       "tenant-list",
		Title:          "Eval B",
		Summary:        "B",
		SourceTaskID:   "task-b",
		SourceReportID: "report-b",
	})
	if err != nil {
		t.Fatalf("CreateCase(caseB) error = %v", err)
	}
	second, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-list",
		SourceCaseID: caseB.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("second eval case reused first ID")
	}

	caseOther, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:       "tenant-other",
		Title:          "Other tenant",
		Summary:        "C",
		SourceTaskID:   "task-c",
		SourceReportID: "report-c",
	})
	if err != nil {
		t.Fatalf("CreateCase(caseOther) error = %v", err)
	}
	if _, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-other",
		SourceCaseID: caseOther.ID,
	}); err != nil {
		t.Fatalf("PromoteCase(other) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-list&limit=1")
	if err != nil {
		t.Fatalf("Get(first page) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var firstPage listEvalCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("Decode(first page) error = %v", err)
	}
	if len(firstPage.EvalCases) != 1 {
		t.Fatalf("len(EvalCases) = %d, want 1", len(firstPage.EvalCases))
	}
	if firstPage.EvalCases[0].EvalCaseID != second.ID {
		t.Fatalf("first page EvalCaseID = %q, want %q", firstPage.EvalCases[0].EvalCaseID, second.ID)
	}
	if !firstPage.HasMore || firstPage.NextOffset == nil || *firstPage.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", firstPage)
	}

	filteredResp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-list&source_task_id=task-a&source_report_id=report-a")
	if err != nil {
		t.Fatalf("Get(filtered) error = %v", err)
	}
	defer filteredResp.Body.Close()
	if filteredResp.StatusCode != http.StatusOK {
		t.Fatalf("filtered StatusCode = %d, want %d", filteredResp.StatusCode, http.StatusOK)
	}

	var filtered listEvalCasesResponse
	if err := json.NewDecoder(filteredResp.Body).Decode(&filtered); err != nil {
		t.Fatalf("Decode(filtered) error = %v", err)
	}
	if len(filtered.EvalCases) != 1 || filtered.EvalCases[0].EvalCaseID != first.ID {
		t.Fatalf("filtered EvalCases = %#v, want only %q", filtered.EvalCases, first.ID)
	}
}

func TestListEvalCasesEndpointRequiresTenantScope(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     casesvc.NewService(),
		EvalCases: evalsvc.NewService(casesvc.NewService(), nil),
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-cases")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

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

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-1")
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

func TestCreateAndGetCaseEndpointWithEvalReportSource(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-case-source", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Source", "Source Eval Source")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-source","title":"Investigate regression","summary":"Follow up failing eval comparison","source_eval_report_id":"` + evalReportID + `","created_by":"operator-eval"}`)
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
		t.Fatalf("Decode(created) error = %v", err)
	}
	if created.SourceEvalReportID != evalReportID {
		t.Fatalf("SourceEvalReportID = %q, want %q", created.SourceEvalReportID, evalReportID)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-eval-case-source")
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
	if got.SourceEvalReportID != evalReportID {
		t.Fatalf("SourceEvalReportID = %q, want %q", got.SourceEvalReportID, evalReportID)
	}
}

func TestCreateAndGetCaseEndpointWithEvalCaseSource(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-case-source-detail", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Case Source", "Source Eval Case Source")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if len(reportItem.BadCases) == 0 {
		t.Fatal("BadCases = 0, want at least one")
	}
	evalCaseID := reportItem.BadCases[0].EvalCaseID

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-source-detail","title":"Investigate precise regression","summary":"Follow up one bad eval case","source_eval_report_id":"` + evalReportID + `","source_eval_case_id":"` + evalCaseID + `","created_by":"operator-eval"}`)
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
		t.Fatalf("Decode(created) error = %v", err)
	}
	if created.SourceEvalCaseID != evalCaseID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", created.SourceEvalCaseID, evalCaseID)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-eval-case-source-detail")
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
	if got.SourceEvalCaseID != evalCaseID {
		t.Fatalf("Get().SourceEvalCaseID = %q, want %q", got.SourceEvalCaseID, evalCaseID)
	}
}

func TestCreateAndGetCaseEndpointWithEvalRunSource(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-run-case-source", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Run Source", "Source Eval Run Source")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if len(reportItem.BadCases) == 0 {
		t.Fatal("BadCases = 0, want at least one")
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalCaseService,
		EvalRuns:  runService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-run-case-source","title":"Investigate eval run","summary":"Follow up one eval run result","source_eval_case_id":"` + reportItem.BadCases[0].EvalCaseID + `","source_eval_run_id":"` + reportItem.RunID + `","created_by":"operator-eval-run"}`)
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
		t.Fatalf("Decode(created) error = %v", err)
	}
	if created.SourceEvalCaseID != reportItem.BadCases[0].EvalCaseID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", created.SourceEvalCaseID, reportItem.BadCases[0].EvalCaseID)
	}
	if created.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("SourceEvalRunID = %q, want %q", created.SourceEvalRunID, reportItem.RunID)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-eval-run-case-source")
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
	if got.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("Get().SourceEvalRunID = %q, want %q", got.SourceEvalRunID, reportItem.RunID)
	}
	if got.SourceEvalCaseID != reportItem.BadCases[0].EvalCaseID {
		t.Fatalf("Get().SourceEvalCaseID = %q, want %q", got.SourceEvalCaseID, reportItem.BadCases[0].EvalCaseID)
	}
}

func TestCreateCaseEndpointReusesOpenCaseBySourceEvalRun(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-run-case-reuse", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Run Reuse", "Source Eval Run Reuse")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if len(reportItem.BadCases) == 0 {
		t.Fatal("BadCases = 0, want at least one")
	}

	existing, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:        "tenant-eval-run-case-reuse",
		Title:           "Existing run-backed follow-up",
		Summary:         "Existing follow-up summary",
		SourceEvalRunID: reportItem.RunID,
		CreatedBy:       "operator-existing-run",
	})
	if err != nil {
		t.Fatalf("CreateCase(existing) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalCaseService,
		EvalRuns:  runService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-run-case-reuse","title":"Investigate eval run again","summary":"Second click should reuse the open run-backed case","source_eval_case_id":"` + reportItem.BadCases[0].EvalCaseID + `","source_eval_run_id":"` + reportItem.RunID + `","created_by":"operator-eval-run"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var reused caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&reused); err != nil {
		t.Fatalf("Decode(reused) error = %v", err)
	}
	if reused.CaseID != existing.ID {
		t.Fatalf("CaseID = %q, want %q", reused.CaseID, existing.ID)
	}
	if reused.SourceEvalRunID != reportItem.RunID {
		t.Fatalf("SourceEvalRunID = %q, want %q", reused.SourceEvalRunID, reportItem.RunID)
	}

	page, err := caseService.ListCases(context.Background(), casesvc.ListFilter{
		TenantID:        "tenant-eval-run-case-reuse",
		SourceEvalRunID: reportItem.RunID,
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want 1", len(page.Cases))
	}
}

func TestCreateAndGetCaseEndpointWithStandaloneEvalCaseSource(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)

	sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-eval-case-standalone",
		Title:    "Source eval case only",
		Summary:  "Promote this one bad case",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(context.Background(), evalsvc.CreateInput{
		TenantID:     "tenant-eval-case-standalone",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalCaseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-standalone","title":"Investigate standalone eval case","summary":"Follow up directly from eval lane","source_eval_case_id":"` + evalCase.ID + `","created_by":"operator-eval"}`)
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
		t.Fatalf("Decode(created) error = %v", err)
	}
	if created.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", created.SourceEvalCaseID, evalCase.ID)
	}
	if created.SourceEvalReportID != "" {
		t.Fatalf("SourceEvalReportID = %q, want empty", created.SourceEvalReportID)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-eval-case-standalone")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", getResp.StatusCode, http.StatusOK)
	}
}

func TestCreateCaseEndpointReusesOpenStandaloneEvalCaseFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)

	sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-eval-case-standalone-reuse",
		Title:    "Source eval case reuse",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	evalCase, _, err := evalCaseService.PromoteCase(context.Background(), evalsvc.CreateInput{
		TenantID:     "tenant-eval-case-standalone-reuse",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	existing, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-eval-case-standalone-reuse",
		Title:            "Existing standalone eval follow-up",
		Summary:          "Existing follow-up summary",
		SourceEvalCaseID: evalCase.ID,
		CreatedBy:        "operator-existing",
	})
	if err != nil {
		t.Fatalf("CreateCase(existing) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalCaseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-standalone-reuse","title":"Investigate standalone eval case","summary":"Follow up directly from eval lane","source_eval_case_id":"` + evalCase.ID + `","created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.CaseID != existing.ID {
		t.Fatalf("CaseID = %q, want %q", got.CaseID, existing.ID)
	}
	if got.SourceEvalCaseID != evalCase.ID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", got.SourceEvalCaseID, evalCase.ID)
	}
}

func TestCreateCaseEndpointReusesOpenEvalReportFollowUp(t *testing.T) {
	reportService, evalReportID := buildEvalReportFixture(t, "tenant-eval-case-reuse", "failed", "failure detail")
	caseService := casesvc.NewService()

	existing, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-case-reuse",
		Title:              "Existing eval follow-up",
		Summary:            "Existing follow-up summary",
		SourceEvalReportID: evalReportID,
		CreatedBy:          "operator-existing",
	})
	if err != nil {
		t.Fatalf("CreateCase(existing) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-reuse","title":"Investigate regression","summary":"Follow up failing eval report","source_eval_report_id":"` + evalReportID + `","created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.CaseID != existing.ID {
		t.Fatalf("CaseID = %q, want %q", got.CaseID, existing.ID)
	}
	if got.SourceEvalReportID != evalReportID {
		t.Fatalf("SourceEvalReportID = %q, want %q", got.SourceEvalReportID, evalReportID)
	}

	page, err := caseService.ListCases(context.Background(), casesvc.ListFilter{
		TenantID:           "tenant-eval-case-reuse",
		Status:             casesvc.StatusOpen,
		SourceEvalReportID: evalReportID,
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 1)
	}
}

func TestCreateCaseEndpointReusesOpenEvalCaseFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-case-reuse-detail", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Reuse Detail", "Source Eval Reuse Detail")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	evalCaseID := reportItem.BadCases[0].EvalCaseID

	existing, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-case-reuse-detail",
		Title:              "Existing eval-case follow-up",
		Summary:            "Existing follow-up summary",
		SourceEvalReportID: evalReportID,
		SourceEvalCaseID:   evalCaseID,
		CreatedBy:          "operator-existing",
	})
	if err != nil {
		t.Fatalf("CreateCase(existing) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-reuse-detail","title":"Investigate one bad case","summary":"Follow up exact bad case","source_eval_report_id":"` + evalReportID + `","source_eval_case_id":"` + evalCaseID + `","created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.CaseID != existing.ID {
		t.Fatalf("CaseID = %q, want %q", got.CaseID, existing.ID)
	}
	if got.SourceEvalCaseID != evalCaseID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", got.SourceEvalCaseID, evalCaseID)
	}
}

func TestCreateCaseEndpointDoesNotReuseGenericEvalReportCaseForEvalCaseFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-case-specific", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Specific", "Source Eval Specific")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	evalCaseID := reportItem.BadCases[0].EvalCaseID

	generic, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-case-specific",
		Title:              "Generic eval-report follow-up",
		Summary:            "Existing report-level follow-up summary",
		SourceEvalReportID: evalReportID,
		CreatedBy:          "operator-existing",
	})
	if err != nil {
		t.Fatalf("CreateCase(generic) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-specific","title":"Investigate one bad case","summary":"Follow up exact bad case","source_eval_report_id":"` + evalReportID + `","source_eval_case_id":"` + evalCaseID + `","created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.CaseID == generic.ID {
		t.Fatalf("CaseID = %q, want a new eval-case-backed follow-up distinct from %q", got.CaseID, generic.ID)
	}
	if got.SourceEvalCaseID != evalCaseID {
		t.Fatalf("SourceEvalCaseID = %q, want %q", got.SourceEvalCaseID, evalCaseID)
	}
}

func TestCreateCaseEndpointDoesNotReuseEvalCaseSpecificCaseForPlainEvalReportFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	evalReportID := materializeEvalRunReport(t, "tenant-eval-report-plain-dedupe", evalsvc.RunStatusFailed, "failure detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Eval Report Plain", "Source Eval Report Plain")
	reportItem, err := reportService.GetEvalReport(context.Background(), evalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	evalCaseID := reportItem.BadCases[0].EvalCaseID

	badCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-report-plain-dedupe",
		Title:              "Existing bad-case follow-up",
		Summary:            "Existing case should not satisfy plain report dedupe",
		SourceEvalReportID: evalReportID,
		SourceEvalCaseID:   evalCaseID,
		CreatedBy:          "operator-existing",
	})
	if err != nil {
		t.Fatalf("CreateCase(badCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-report-plain-dedupe","title":"Investigate report-level regression","summary":"Create plain report follow-up","source_eval_report_id":"` + evalReportID + `","created_by":"operator-report"}`)
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
	if created.CaseID == badCase.ID {
		t.Fatal("CreateCase() reused eval-case-specific case, want plain report follow-up")
	}
	if created.SourceEvalCaseID != "" {
		t.Fatalf("SourceEvalCaseID = %q, want empty", created.SourceEvalCaseID)
	}

	page, err := caseService.ListCases(context.Background(), casesvc.ListFilter{
		TenantID:           "tenant-eval-report-plain-dedupe",
		Status:             casesvc.StatusOpen,
		SourceEvalReportID: evalReportID,
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 2 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 2)
	}
}

func TestCreateCaseEndpointDoesNotReuseCompareOriginEvalCaseForPlainEvalCaseFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-specific", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Specific Left", "Source Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-specific", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Specific Right", "Source Right")
	rightReport, err := reportService.GetEvalReport(context.Background(), rightEvalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	evalCaseID := rightReport.BadCases[0].EvalCaseID

	compareCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-case-compare-specific",
		Title:              "Compare eval-case follow-up",
		Summary:            "Compare-derived case should not satisfy plain eval-case dedupe",
		SourceEvalReportID: rightEvalReportID,
		SourceEvalCaseID:   evalCaseID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftEvalReportID,
			RightEvalReportID: rightEvalReportID,
			SelectedSide:      "right",
		},
		CreatedBy: "operator-compare",
	})
	if err != nil {
		t.Fatalf("CreateCase(compareCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-compare-specific","title":"Plain eval-case follow-up","summary":"Create plain eval-case follow-up","source_eval_report_id":"` + rightEvalReportID + `","source_eval_case_id":"` + evalCaseID + `","created_by":"operator-plain"}`)
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
	if created.CaseID == compareCase.ID {
		t.Fatalf("CaseID = %q, want a new plain eval-case-backed follow-up distinct from compare-origin case %q", created.CaseID, compareCase.ID)
	}
	if created.CompareOrigin != nil {
		t.Fatal("CreateCase() reused compare-origin case, want plain eval-case follow-up")
	}
}

func TestCreateCaseEndpointRejectsEvalCaseOutsideEvalReport(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-mismatch", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Mismatch Left", "Source Mismatch Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-mismatch", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Mismatch Right", "Source Mismatch Right")
	rightReport, err := reportService.GetEvalReport(context.Background(), rightEvalReportID)
	if err != nil {
		t.Fatalf("GetEvalReport(right) error = %v", err)
	}
	evalCaseID := rightReport.BadCases[0].EvalCaseID

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalCases:   evalCaseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-mismatch","title":"Invalid precise follow-up","summary":"This bad case does not belong to the selected report","source_eval_report_id":"` + leftEvalReportID + `","source_eval_case_id":"` + evalCaseID + `","created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestCreateCaseEndpointReusesOpenCompareOriginFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-distinct", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Left", "Source Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-distinct", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Right", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := `{"tenant_id":"tenant-eval-case-compare-distinct","title":"Investigate compare regression","summary":"Follow up selected compare side","source_eval_report_id":"` + rightEvalReportID + `","compare_origin":{"left_eval_report_id":"` + leftEvalReportID + `","right_eval_report_id":"` + rightEvalReportID + `","selected_side":"right"},"created_by":"operator-eval"}`
	firstResp, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Post(first) error = %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("first StatusCode = %d, want %d", firstResp.StatusCode, http.StatusCreated)
	}
	var first caseResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&first); err != nil {
		t.Fatalf("Decode(first) error = %v", err)
	}

	secondResp, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Post(second) error = %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("second StatusCode = %d, want %d", secondResp.StatusCode, http.StatusOK)
	}
	var second caseResponse
	if err := json.NewDecoder(secondResp.Body).Decode(&second); err != nil {
		t.Fatalf("Decode(second) error = %v", err)
	}
	if second.CaseID != first.CaseID {
		t.Fatalf("compare-origin case creation returned %q, want reused case %q", second.CaseID, first.CaseID)
	}
}

func TestCreateCaseEndpointKeepsDistinctCompareOriginFollowUpsForDifferentComparisons(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-distinct", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Left", "Source Left")
	altLeftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-distinct", evalsvc.RunStatusSucceeded, "alt left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Alt Left", "Source Alt Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-distinct", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Right", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	firstBody := `{"tenant_id":"tenant-eval-case-compare-distinct","title":"Investigate compare regression","summary":"Follow up selected compare side","source_eval_report_id":"` + rightEvalReportID + `","compare_origin":{"left_eval_report_id":"` + leftEvalReportID + `","right_eval_report_id":"` + rightEvalReportID + `","selected_side":"right"},"created_by":"operator-eval"}`
	firstResp, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewBufferString(firstBody))
	if err != nil {
		t.Fatalf("Post(first) error = %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("first StatusCode = %d, want %d", firstResp.StatusCode, http.StatusCreated)
	}
	var first caseResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&first); err != nil {
		t.Fatalf("Decode(first) error = %v", err)
	}

	secondBody := `{"tenant_id":"tenant-eval-case-compare-distinct","title":"Investigate different compare regression","summary":"Follow up a different comparison","source_eval_report_id":"` + rightEvalReportID + `","compare_origin":{"left_eval_report_id":"` + altLeftEvalReportID + `","right_eval_report_id":"` + rightEvalReportID + `","selected_side":"right"},"created_by":"operator-eval"}`
	secondResp, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewBufferString(secondBody))
	if err != nil {
		t.Fatalf("Post(second) error = %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusCreated {
		t.Fatalf("second StatusCode = %d, want %d", secondResp.StatusCode, http.StatusCreated)
	}
	var second caseResponse
	if err := json.NewDecoder(secondResp.Body).Decode(&second); err != nil {
		t.Fatalf("Decode(second) error = %v", err)
	}
	if second.CaseID == first.CaseID {
		t.Fatal("different compare-origin lineage reused the first case, want a distinct compare-derived follow-up")
	}
}

func TestCreateCaseEndpointDoesNotReuseCompareOriginCaseForPlainEvalFollowUp(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-mixed-dedupe", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Mixed Left", "Source Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-mixed-dedupe", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Mixed Right", "Source Right")

	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-eval-case-mixed-dedupe",
		Title:              "Compare-origin follow-up",
		Summary:            "Compare-origin case should not satisfy plain dedupe",
		SourceEvalReportID: rightEvalReportID,
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  leftEvalReportID,
			RightEvalReportID: rightEvalReportID,
			SelectedSide:      "right",
		},
		CreatedBy: "operator-compare",
	}); err != nil {
		t.Fatalf("CreateCase(compareOrigin) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-mixed-dedupe","title":"Plain follow-up","summary":"Create plain eval-report follow-up","source_eval_report_id":"` + rightEvalReportID + `","created_by":"operator-plain"}`)
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
	if created.CompareOrigin != nil {
		t.Fatal("CreateCase() reused compare-origin case, want plain follow-up case")
	}

	page, err := caseService.ListCases(context.Background(), casesvc.ListFilter{
		TenantID:           "tenant-eval-case-mixed-dedupe",
		Status:             casesvc.StatusOpen,
		SourceEvalReportID: rightEvalReportID,
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 2 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 2)
	}
}

func TestCreateAndGetCaseEndpointWithCompareOrigin(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Left", "Source Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Right", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-compare","title":"Investigate compare regression","summary":"Follow up selected compare side","source_eval_report_id":"` + rightEvalReportID + `","compare_origin":{"left_eval_report_id":"` + leftEvalReportID + `","right_eval_report_id":"` + rightEvalReportID + `","selected_side":"right"},"created_by":"operator-eval"}`)
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
		t.Fatalf("Decode(created) error = %v", err)
	}
	if created.CompareOrigin == nil {
		t.Fatal("CompareOrigin is nil")
	}
	if created.CompareOrigin.LeftEvalReportID != leftEvalReportID {
		t.Fatalf("CompareOrigin.LeftEvalReportID = %q, want %q", created.CompareOrigin.LeftEvalReportID, leftEvalReportID)
	}
	if created.CompareOrigin.RightEvalReportID != rightEvalReportID {
		t.Fatalf("CompareOrigin.RightEvalReportID = %q, want %q", created.CompareOrigin.RightEvalReportID, rightEvalReportID)
	}
	if created.CompareOrigin.SelectedSide != "right" {
		t.Fatalf("CompareOrigin.SelectedSide = %q, want %q", created.CompareOrigin.SelectedSide, "right")
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.CaseID + "?tenant_id=tenant-eval-case-compare")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	var got caseResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode(get) error = %v", err)
	}
	if got.CompareOrigin == nil {
		t.Fatal("Get().CompareOrigin is nil")
	}
	if got.CompareOrigin.LeftEvalReportID != leftEvalReportID {
		t.Fatalf("Get().CompareOrigin.LeftEvalReportID = %q, want %q", got.CompareOrigin.LeftEvalReportID, leftEvalReportID)
	}
	if got.CompareOrigin.RightEvalReportID != rightEvalReportID {
		t.Fatalf("Get().CompareOrigin.RightEvalReportID = %q, want %q", got.CompareOrigin.RightEvalReportID, rightEvalReportID)
	}
	if got.CompareOrigin.SelectedSide != "right" {
		t.Fatalf("Get().CompareOrigin.SelectedSide = %q, want %q", got.CompareOrigin.SelectedSide, "right")
	}
}

func TestCreateCaseRejectsCompareOriginMixedWithTaskOrReportSources(t *testing.T) {
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)
	leftEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-mixed", evalsvc.RunStatusSucceeded, "left detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Mixed Left", "Source Left")
	rightEvalReportID := materializeEvalRunReport(t, "tenant-eval-case-compare-mixed", evalsvc.RunStatusFailed, "right detail", caseService, evalCaseService, datasetService, runService, reportService, "Dataset Compare Mixed Right", "Source Right")

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-eval-case-compare-mixed","title":"Investigate compare regression","summary":"Follow up selected compare side","source_task_id":"task-mixed","source_eval_report_id":"` + rightEvalReportID + `","compare_origin":{"left_eval_report_id":"` + leftEvalReportID + `","right_eval_report_id":"` + rightEvalReportID + `","selected_side":"right"},"created_by":"operator-eval"}`)
	resp, err := http.Post(server.URL+"/api/v1/cases", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_case" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_case")
	}
	if got.Message != "compare_origin cannot be combined with source_task_id, source_report_id, source_eval_case_id, or source_eval_run_id" {
		t.Fatalf("Message = %q, want %q", got.Message, "compare_origin cannot be combined with source_task_id, source_report_id, source_eval_case_id, or source_eval_run_id")
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

func TestListCasesEndpointSupportsFiltersAndOffset(t *testing.T) {
	caseService := casesvc.NewService()

	first, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "First case",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	second, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:       "tenant-1",
		Title:          "Second case",
		SourceTaskID:   "task-2",
		SourceReportID: "report-2",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-2",
		Title:    "Third case",
	}); err != nil {
		t.Fatalf("CreateCase(third) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var firstPage listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("Decode(firstPage) error = %v", err)
	}
	if len(firstPage.Cases) != 1 {
		t.Fatalf("len(firstPage.Cases) = %d, want %d", len(firstPage.Cases), 1)
	}
	if firstPage.Cases[0].CaseID != second.ID {
		t.Fatalf("firstPage.Cases[0].CaseID = %q, want %q", firstPage.Cases[0].CaseID, second.ID)
	}
	if !firstPage.HasMore {
		t.Fatal("firstPage.HasMore = false, want true")
	}

	nextResp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&limit=1&offset=1")
	if err != nil {
		t.Fatalf("Get(offset=1) error = %v", err)
	}
	defer nextResp.Body.Close()

	var nextPage listCasesResponse
	if err := json.NewDecoder(nextResp.Body).Decode(&nextPage); err != nil {
		t.Fatalf("Decode(nextPage) error = %v", err)
	}
	if len(nextPage.Cases) != 1 {
		t.Fatalf("len(nextPage.Cases) = %d, want %d", len(nextPage.Cases), 1)
	}
	if nextPage.Cases[0].CaseID != first.ID {
		t.Fatalf("nextPage.Cases[0].CaseID = %q, want %q", nextPage.Cases[0].CaseID, first.ID)
	}
}

func TestListCasesEndpointSupportsAssignedToFilter(t *testing.T) {
	caseService := casesvc.NewService()

	first, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Mine",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	second, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Other",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), first, "cases-operator"); err != nil {
		t.Fatalf("AssignCase(first) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), second, "other-operator"); err != nil {
		t.Fatalf("AssignCase(second) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&status=open&assigned_to=cases-operator&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].CaseID != first.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, first.ID)
	}
}

func TestListCasesEndpointSupportsUnassignedOnlyFilter(t *testing.T) {
	caseService := casesvc.NewService()

	unassigned, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Unassigned",
	})
	if err != nil {
		t.Fatalf("CreateCase(unassigned) error = %v", err)
	}
	assigned, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Assigned",
	})
	if err != nil {
		t.Fatalf("CreateCase(assigned) error = %v", err)
	}
	if _, err := caseService.AssignCase(context.Background(), assigned, "cases-operator"); err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&status=open&unassigned_only=true&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].CaseID != unassigned.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, unassigned.ID)
	}
}

func TestListCasesEndpointSupportsEvalReportSourceFilter(t *testing.T) {
	caseService := casesvc.NewService()

	evalBacked, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Eval-backed case",
		SourceEvalReportID: "eval-report-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(evalBacked) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Other eval-backed case",
		SourceEvalReportID: "eval-report-2",
	}); err != nil {
		t.Fatalf("CreateCase(otherEvalBacked) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Non eval-backed case",
	}); err != nil {
		t.Fatalf("CreateCase(nonEvalBacked) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&source_eval_report_id=eval-report-1&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].CaseID != evalBacked.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, evalBacked.ID)
	}
	if page.Cases[0].SourceEvalReportID != "eval-report-1" {
		t.Fatalf("page.Cases[0].SourceEvalReportID = %q, want %q", page.Cases[0].SourceEvalReportID, "eval-report-1")
	}
}

func TestListCasesEndpointSupportsEvalDatasetSourceFilter(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalCaseService := evalsvc.NewService(caseService, nil)
	datasetService := evalsvc.NewDatasetService(evalCaseService)
	runService := evalsvc.NewRunService(datasetService)
	reportService := evalsvc.NewEvalReportServiceWithDependencies(nil, runService)

	sourceCaseA, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-cases",
		Title:    "Dataset source A",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCaseA) error = %v", err)
	}
	evalCaseA, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-cases",
		SourceCaseID: sourceCaseA.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalCaseA) error = %v", err)
	}
	sourceCaseB, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset-cases",
		Title:    "Dataset source B",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCaseB) error = %v", err)
	}
	evalCaseB, _, err := evalCaseService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-dataset-cases",
		SourceCaseID: sourceCaseB.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalCaseB) error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, evalsvc.CreateDatasetInput{
		TenantID:    "tenant-dataset-cases",
		Name:        "Dataset-backed queue",
		EvalCaseIDs: []string{evalCaseA.ID, evalCaseB.ID},
		CreatedBy:   "operator-dataset",
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, evalsvc.PublishDatasetInput{TenantID: "tenant-dataset-cases"}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	runA, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-dataset-cases",
		DatasetID: dataset.ID,
		CreatedBy: "operator-dataset",
	})
	if err != nil {
		t.Fatalf("CreateRun(runA) error = %v", err)
	}
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(runA) error = %v", err)
	}
	if _, err := runService.MarkRunFailed(ctx, runA.ID, "first run failed"); err != nil {
		t.Fatalf("MarkRunFailed(runA) error = %v", err)
	}
	reportA, err := reportService.MaterializeRunReport(ctx, runA.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport(runA) error = %v", err)
	}

	runB, err := runService.CreateRun(ctx, evalsvc.CreateRunInput{
		TenantID:  "tenant-dataset-cases",
		DatasetID: dataset.ID,
		CreatedBy: "operator-dataset",
	})
	if err != nil {
		t.Fatalf("CreateRun(runB) error = %v", err)
	}
	if _, err := runService.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(runB) error = %v", err)
	}
	if _, err := runService.MarkRunFailed(ctx, runB.ID, "second run failed"); err != nil {
		t.Fatalf("MarkRunFailed(runB) error = %v", err)
	}
	reportB, err := reportService.MaterializeRunReport(ctx, runB.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport(runB) error = %v", err)
	}

	firstCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-cases",
		Title:              "First dataset follow-up",
		SourceEvalReportID: reportA.ID,
	})
	if err != nil {
		t.Fatalf("CreateCase(firstCase) error = %v", err)
	}
	secondCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-cases",
		Title:              "Second dataset follow-up",
		SourceEvalReportID: reportB.ID,
	})
	if err != nil {
		t.Fatalf("CreateCase(secondCase) error = %v", err)
	}
	if _, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-dataset-cases",
		Title:              "Other dataset follow-up",
		SourceEvalReportID: "eval-report-other",
	}); err != nil {
		t.Fatalf("CreateCase(otherCase) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:       caseService,
		EvalReports: reportService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-dataset-cases&source_eval_dataset_id=" + dataset.ID + "&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 2 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 2)
	}
	gotIDs := []string{page.Cases[0].CaseID, page.Cases[1].CaseID}
	if !(gotIDs[0] == secondCase.ID && gotIDs[1] == firstCase.ID) {
		t.Fatalf("got IDs = %#v, want [%q %q]", gotIDs, secondCase.ID, firstCase.ID)
	}
}

func TestListCasesEndpointSupportsEvalBackedOnlyFilter(t *testing.T) {
	caseService := casesvc.NewService()

	firstEvalBacked, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "First eval-backed case",
		SourceEvalReportID: "eval-report-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(firstEvalBacked) error = %v", err)
	}
	secondEvalBacked, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Second eval-backed case",
		SourceEvalReportID: "eval-report-2",
	})
	if err != nil {
		t.Fatalf("CreateCase(secondEvalBacked) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:       "tenant-1",
		Title:          "Report-backed case",
		SourceReportID: "report-1",
	}); err != nil {
		t.Fatalf("CreateCase(reportBacked) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&eval_backed_only=true&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 2 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 2)
	}
	if page.Cases[0].CaseID != secondEvalBacked.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, secondEvalBacked.ID)
	}
	if page.Cases[1].CaseID != firstEvalBacked.ID {
		t.Fatalf("page.Cases[1].CaseID = %q, want %q", page.Cases[1].CaseID, firstEvalBacked.ID)
	}
}

func TestListCasesEndpointSupportsEvalRunSourceFilter(t *testing.T) {
	caseService := casesvc.NewService()

	runBackedA, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-1",
		Title:            "Run-backed case A",
		SourceEvalRunID:  "eval-run-1",
		SourceEvalCaseID: "eval-case-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(runBackedA) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-1",
		Title:            "Run-backed case B",
		SourceEvalRunID:  "eval-run-2",
		SourceEvalCaseID: "eval-case-2",
	}); err != nil {
		t.Fatalf("CreateCase(runBackedB) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Manual case",
	}); err != nil {
		t.Fatalf("CreateCase(manual) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&source_eval_run_id=eval-run-1&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].CaseID != runBackedA.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, runBackedA.ID)
	}
	if page.Cases[0].SourceEvalRunID != "eval-run-1" {
		t.Fatalf("page.Cases[0].SourceEvalRunID = %q, want %q", page.Cases[0].SourceEvalRunID, "eval-run-1")
	}
}

func TestListCasesEndpointSupportsRunBackedOnlyFilter(t *testing.T) {
	caseService := casesvc.NewService()

	firstRunBacked, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:        "tenant-1",
		Title:           "First run-backed case",
		SourceEvalRunID: "eval-run-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(firstRunBacked) error = %v", err)
	}
	secondRunBacked, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:        "tenant-1",
		Title:           "Second run-backed case",
		SourceEvalRunID: "eval-run-2",
	})
	if err != nil {
		t.Fatalf("CreateCase(secondRunBacked) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Eval report-backed case",
		SourceEvalReportID: "eval-report-1",
	}); err != nil {
		t.Fatalf("CreateCase(evalBacked) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&run_backed_only=true&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 2 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 2)
	}
	if page.Cases[0].CaseID != secondRunBacked.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, secondRunBacked.ID)
	}
	if page.Cases[1].CaseID != firstRunBacked.ID {
		t.Fatalf("page.Cases[1].CaseID = %q, want %q", page.Cases[1].CaseID, firstRunBacked.ID)
	}
	for _, item := range page.Cases {
		if item.SourceEvalRunID == "" {
			t.Fatal("run_backed_only returned a case without source_eval_run_id")
		}
	}
}

func TestListCasesEndpointSupportsCompareOriginOnlyFilter(t *testing.T) {
	caseService := casesvc.NewService()

	compareDerived, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Compare-derived case",
		SourceEvalReportID: "eval-report-compare-1",
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  "eval-report-compare-1",
			RightEvalReportID: "eval-report-compare-2",
			SelectedSide:      "left",
		},
	})
	if err != nil {
		t.Fatalf("CreateCase(compareDerived) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Eval-backed only case",
		SourceEvalReportID: "eval-report-compare-3",
	}); err != nil {
		t.Fatalf("CreateCase(evalBackedOnly) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Manual case",
	}); err != nil {
		t.Fatalf("CreateCase(manual) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&compare_origin_only=true&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].CaseID != compareDerived.ID {
		t.Fatalf("page.Cases[0].CaseID = %q, want %q", page.Cases[0].CaseID, compareDerived.ID)
	}
	if page.Cases[0].CompareOrigin == nil || page.Cases[0].CompareOrigin.SelectedSide != "left" {
		t.Fatalf("page.Cases[0].CompareOrigin = %#v, want selected_side=left", page.Cases[0].CompareOrigin)
	}
}

func TestListCasesEndpointRejectsInvalidCompareOriginOnly(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&compare_origin_only=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_query" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_query")
	}
}

func TestListCasesEndpointRejectsInvalidRunBackedOnly(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&run_backed_only=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_query" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_query")
	}
}

func TestListCasesEndpointSupportsEvalReportSourceFilterPagination(t *testing.T) {
	caseService := casesvc.NewService()

	for i := 0; i < 6; i++ {
		if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:           "tenant-1",
			Title:              "Eval-backed case",
			SourceEvalReportID: "eval-report-many",
		}); err != nil {
			t.Fatalf("CreateCase(%d) error = %v", i, err)
		}
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: caseService,
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&source_eval_report_id=eval-report-many&limit=5")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var page listCasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(page.Cases) != 5 {
		t.Fatalf("len(page.Cases) = %d, want %d", len(page.Cases), 5)
	}
	if !page.HasMore {
		t.Fatal("page.HasMore = false, want true")
	}
	if page.NextOffset == nil || *page.NextOffset != 5 {
		t.Fatalf("page.NextOffset = %v, want 5", page.NextOffset)
	}
}

func TestListCasesEndpointRejectsInvalidUnassignedOnly(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?tenant_id=tenant-1&unassigned_only=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_query" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_query")
	}
}

func TestListCasesEndpointRejectsInvalidOffset(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases?offset=-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_query" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_query")
	}
}

func TestListCasesEndpointRequiresTenantID(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_query" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_query")
	}
}

func TestGetCaseEndpointRequiresTenantID(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Tenant-scoped case",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases/" + created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetCaseEndpointFailsClosedForWrongTenant(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-a",
		Title:    "Tenant-scoped case",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/cases/" + created.ID + "?tenant_id=tenant-b")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestCreateAndGetCaseEndpointSupportsReportOnlySource(t *testing.T) {
	reportService := report.NewService()
	caseService := casesvc.NewService()

	task := workflow.Task{
		ID:        "task-report-only",
		RequestID: "req-report-only",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Status:    workflow.StatusSucceeded,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		CreatedAt: time.Unix(1700003000, 0).UTC(),
		UpdatedAt: time.Unix(1700003001, 0).UTC(),
	}
	reportID, err := reportService.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated report-only case source",
	})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Reports: reportService,
		Cases:   caseService,
	}))
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","title":"Report only case","source_report_id":"` + reportID + `"}`)
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
	if created.SourceTaskID != "" {
		t.Fatalf("SourceTaskID = %q, want empty", created.SourceTaskID)
	}
	if created.SourceReportID != reportID {
		t.Fatalf("SourceReportID = %q, want %q", created.SourceReportID, reportID)
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

func TestCloseCaseEndpoint(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case to close",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	body := bytes.NewBufferString(`{"closed_by":"operator-2"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/close?tenant_id=tenant-1", body)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Status != casesvc.StatusClosed {
		t.Fatalf("Status = %q, want %q", got.Status, casesvc.StatusClosed)
	}
	if got.ClosedBy != "operator-2" {
		t.Fatalf("ClosedBy = %q, want %q", got.ClosedBy, "operator-2")
	}
}

func TestCloseCaseEndpointRejectsInvalidState(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case to close twice",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/close?tenant_id=tenant-1", bytes.NewBufferString(`{"closed_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestAssignCaseEndpoint(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case to assign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/assign?tenant_id=tenant-1", bytes.NewBufferString(`{"assigned_to":"owner-1"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.AssignedTo != "owner-1" {
		t.Fatalf("AssignedTo = %q, want %q", got.AssignedTo, "owner-1")
	}
	if got.AssignedAt == "" {
		t.Fatal("AssignedAt is empty")
	}
}

func TestAssignCaseEndpointRejectsClosedCase(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Closed before assign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/assign?tenant_id=tenant-1", bytes.NewBufferString(`{"assigned_to":"owner-1"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestUnassignCaseEndpoint(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case to unassign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	assigned, err := caseService.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+assigned.ID+"/unassign?tenant_id=tenant-1", bytes.NewBufferString(`{"unassigned_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.AssignedTo != "" {
		t.Fatalf("AssignedTo = %q, want empty", got.AssignedTo)
	}
	if got.AssignedAt != "" {
		t.Fatalf("AssignedAt = %q, want empty", got.AssignedAt)
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + assigned.ID + "?tenant_id=tenant-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	var reloaded caseResponse
	if err := json.NewDecoder(getResp.Body).Decode(&reloaded); err != nil {
		t.Fatalf("Decode(reloaded) error = %v", err)
	}
	if len(reloaded.Notes) != 1 {
		t.Fatalf("len(reloaded.Notes) = %d, want %d", len(reloaded.Notes), 1)
	}
	if reloaded.Notes[0].Body != "case returned to queue by operator-2" {
		t.Fatalf("reloaded.Notes[0].Body = %q, want %q", reloaded.Notes[0].Body, "case returned to queue by operator-2")
	}
	if reloaded.Notes[0].CreatedBy != "operator-2" {
		t.Fatalf("reloaded.Notes[0].CreatedBy = %q, want %q", reloaded.Notes[0].CreatedBy, "operator-2")
	}
}

func TestUnassignCaseEndpointRejectsClosedCase(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Closed before unassign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	assigned, err := caseService.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), assigned.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+assigned.ID+"/unassign?tenant_id=tenant-1", bytes.NewBufferString(`{"unassigned_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestUnassignCaseEndpointRejectsAlreadyUnassignedCase(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Already unassigned",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/unassign?tenant_id=tenant-1", bytes.NewBufferString(`{"unassigned_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestReopenCaseEndpoint(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case to reopen",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/reopen?tenant_id=tenant-1", bytes.NewBufferString(`{"reopened_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got caseResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Status != casesvc.StatusOpen {
		t.Fatalf("Status = %q, want %q", got.Status, casesvc.StatusOpen)
	}
	if got.ClosedBy != "" {
		t.Fatalf("ClosedBy = %q, want empty", got.ClosedBy)
	}

	reloaded, err := caseService.GetCase(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetCase() error = %v", err)
	}
	notes, err := caseService.ListCaseNotes(context.Background(), reloaded.ID, 10)
	if err != nil {
		t.Fatalf("ListCaseNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(notes) = %d, want %d", len(notes), 1)
	}
	if notes[0].Body != "case reopened by operator-2" {
		t.Fatalf("notes[0].Body = %q, want %q", notes[0].Body, "case reopened by operator-2")
	}
}

func TestReopenCaseEndpointRejectsOpenCase(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Already open",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/reopen?tenant_id=tenant-1", bytes.NewBufferString(`{"reopened_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestAssignCaseEndpointRejectsStaleWrite(t *testing.T) {
	store := &staleAssignStore{
		item: casesvc.Case{
			ID:        "case-stale-1",
			TenantID:  "tenant-1",
			Status:    casesvc.StatusOpen,
			Title:     "Stale assign",
			CreatedBy: "operator-1",
			CreatedAt: time.Unix(1700000000, 0).UTC(),
			UpdatedAt: time.Unix(1700000000, 0).UTC(),
		},
	}
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: casesvc.NewServiceWithStore(store),
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/case-stale-1/assign?tenant_id=tenant-1", bytes.NewBufferString(`{"assigned_to":"owner-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "case_conflict" {
		t.Fatalf("Code = %q, want %q", got.Code, "case_conflict")
	}
}

func TestUnassignCaseEndpointRejectsStaleWrite(t *testing.T) {
	store := &staleAssignStore{
		item: casesvc.Case{
			ID:         "case-stale-unassign-1",
			TenantID:   "tenant-1",
			Status:     casesvc.StatusOpen,
			Title:      "Stale unassign",
			CreatedBy:  "operator-1",
			AssignedTo: "owner-1",
			AssignedAt: time.Unix(1700000000, 0).UTC(),
			CreatedAt:  time.Unix(1700000000, 0).UTC(),
			UpdatedAt:  time.Unix(1700000000, 0).UTC(),
		},
	}
	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases: casesvc.NewServiceWithStore(store),
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/case-stale-unassign-1/unassign?tenant_id=tenant-1", bytes.NewBufferString(`{"unassigned_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "case_conflict" {
		t.Fatalf("Code = %q, want %q", got.Code, "case_conflict")
	}
}

func TestAddCaseNoteEndpointAndGetCaseIncludesNotes(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case with note",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/notes?tenant_id=tenant-1", bytes.NewBufferString(`{"body":"note body","created_by":"operator-a"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var note caseNoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		t.Fatalf("Decode(note) error = %v", err)
	}
	if note.Body != "note body" {
		t.Fatalf("note.Body = %q, want %q", note.Body, "note body")
	}

	getResp, err := http.Get(server.URL + "/api/v1/cases/" + created.ID + "?tenant_id=tenant-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer getResp.Body.Close()

	var got caseResponse
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode(case) error = %v", err)
	}
	if len(got.Notes) != 1 {
		t.Fatalf("len(got.Notes) = %d, want %d", len(got.Notes), 1)
	}
	if got.Notes[0].Body != "note body" {
		t.Fatalf("got.Notes[0].Body = %q, want %q", got.Notes[0].Body, "note body")
	}
}

func TestAddCaseNoteEndpointRejectsEmptyBody(t *testing.T) {
	caseService := casesvc.NewService()
	created, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Case with empty note",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Cases: caseService}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/cases/"+created.ID+"/notes?tenant_id=tenant-1", bytes.NewBufferString(`{"body":"   "}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "invalid_note" {
		t.Fatalf("Code = %q, want %q", got.Code, "invalid_note")
	}
}

type staleAssignStore struct {
	item casesvc.Case
}

func (s *staleAssignStore) Save(_ context.Context, item casesvc.Case) (casesvc.Case, error) {
	s.item = item
	return item, nil
}

func (s *staleAssignStore) Get(_ context.Context, caseID string) (casesvc.Case, error) {
	if s.item.ID != caseID {
		return casesvc.Case{}, casesvc.ErrCaseNotFound
	}
	return s.item, nil
}

func (s *staleAssignStore) List(_ context.Context, _ casesvc.ListFilter) (casesvc.ListPage, error) {
	return casesvc.ListPage{Cases: []casesvc.Case{s.item}}, nil
}

func (s *staleAssignStore) FindOpenByCompareOrigin(_ context.Context, _ string, _ string, _ casesvc.CompareOrigin) (casesvc.Case, bool, error) {
	return casesvc.Case{}, false, nil
}

func (s *staleAssignStore) SummarizeBySourceEvalReportIDs(_ context.Context, _ string, reportIDs []string) (map[string]casesvc.EvalReportFollowUpSummary, error) {
	summaries := make(map[string]casesvc.EvalReportFollowUpSummary, len(reportIDs))
	for _, reportID := range reportIDs {
		summaries[reportID] = casesvc.EvalReportFollowUpSummary{SourceEvalReportID: reportID}
	}
	return summaries, nil
}

func (s *staleAssignStore) SummarizeBySourceEvalCaseIDs(_ context.Context, _ string, evalCaseIDs []string) (map[string]casesvc.EvalCaseFollowUpSummary, error) {
	summaries := make(map[string]casesvc.EvalCaseFollowUpSummary, len(evalCaseIDs))
	for _, evalCaseID := range evalCaseIDs {
		summaries[evalCaseID] = casesvc.EvalCaseFollowUpSummary{SourceEvalCaseID: evalCaseID}
	}
	return summaries, nil
}

func (s *staleAssignStore) SummarizeCompareOriginBySourceEvalReportIDs(_ context.Context, _ string, reportIDs []string) (map[string]casesvc.EvalReportCompareFollowUpSummary, error) {
	summaries := make(map[string]casesvc.EvalReportCompareFollowUpSummary, len(reportIDs))
	for _, reportID := range reportIDs {
		summaries[reportID] = casesvc.EvalReportCompareFollowUpSummary{SourceEvalReportID: reportID}
	}
	return summaries, nil
}

func (s *staleAssignStore) AppendNote(_ context.Context, note casesvc.Note) (casesvc.Note, error) {
	return note, nil
}

func (s *staleAssignStore) ListNotes(_ context.Context, caseID string, limit int) ([]casesvc.Note, error) {
	return []casesvc.Note{}, nil
}

func (s *staleAssignStore) Assign(_ context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (casesvc.Case, error) {
	if s.item.ID != caseID {
		return casesvc.Case{}, casesvc.ErrCaseNotFound
	}
	return casesvc.Case{}, casesvc.ErrCaseConflict
}

func (s *staleAssignStore) Unassign(_ context.Context, caseID string, unassignedBy string, unassignedAt time.Time, expectedUpdatedAt time.Time) (casesvc.Case, error) {
	if s.item.ID != caseID {
		return casesvc.Case{}, casesvc.ErrCaseNotFound
	}
	return casesvc.Case{}, casesvc.ErrCaseConflict
}

func (s *staleAssignStore) Close(_ context.Context, caseID string, closedBy string, closedAt time.Time) (casesvc.Case, error) {
	if s.item.ID != caseID {
		return casesvc.Case{}, casesvc.ErrCaseNotFound
	}
	return casesvc.Case{}, casesvc.ErrInvalidCaseState
}

func (s *staleAssignStore) Reopen(_ context.Context, caseID string, reopenedBy string, reopenedAt time.Time) (casesvc.Case, error) {
	if s.item.ID != caseID {
		return casesvc.Case{}, casesvc.ErrCaseNotFound
	}
	return casesvc.Case{}, casesvc.ErrInvalidCaseState
}

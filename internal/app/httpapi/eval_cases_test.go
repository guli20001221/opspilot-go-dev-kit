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
	if created.PreferredFollowUpAction.Mode != "create" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", created.PreferredFollowUpAction.Mode, "create")
	}
	if created.PreferredFollowUpAction.SourceEvalCaseID != created.EvalCaseID {
		t.Fatalf("PreferredFollowUpAction.SourceEvalCaseID = %q, want %q", created.PreferredFollowUpAction.SourceEvalCaseID, created.EvalCaseID)
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

	_, err = caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:         "tenant-1",
		Title:            "Follow-up from eval case",
		SourceEvalCaseID: first.EvalCaseID,
	})
	if err != nil {
		t.Fatalf("CreateCase(follow-up) error = %v", err)
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
	if second.FollowUpCaseCount != 1 {
		t.Fatalf("second.FollowUpCaseCount = %d, want %d", second.FollowUpCaseCount, 1)
	}
	if second.OpenFollowUpCaseCount != 1 {
		t.Fatalf("second.OpenFollowUpCaseCount = %d, want %d", second.OpenFollowUpCaseCount, 1)
	}
	if second.LatestFollowUpCaseID == "" {
		t.Fatal("second.LatestFollowUpCaseID is empty")
	}
	if second.LatestFollowUpCaseStatus != string(casesvc.StatusOpen) {
		t.Fatalf("second.LatestFollowUpCaseStatus = %q, want %q", second.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if second.PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("second.PreferredFollowUpAction.Mode = %q, want %q", second.PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if second.PreferredFollowUpAction.CaseID != second.LatestFollowUpCaseID {
		t.Fatalf("second.PreferredFollowUpAction.CaseID = %q, want %q", second.PreferredFollowUpAction.CaseID, second.LatestFollowUpCaseID)
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

func TestListEvalCasesEndpointSupportsNeedsFollowUpFilter(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := evalsvc.NewService(caseService, nil)

	sourceOpen, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-needs-follow-up",
		Title:    "Open follow-up source",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceOpen) error = %v", err)
	}
	evalOpen, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-needs-follow-up",
		SourceCaseID: sourceOpen.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalOpen) error = %v", err)
	}
	if _, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-needs-follow-up",
		Title:              "Open linked case",
		SourceEvalCaseID:   evalOpen.ID,
		SourceEvalReportID: "eval-report-open",
	}); err != nil {
		t.Fatalf("CreateCase(open linked case) error = %v", err)
	}

	sourceClosed, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-needs-follow-up",
		Title:    "Closed follow-up source",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceClosed) error = %v", err)
	}
	evalClosed, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-needs-follow-up",
		SourceCaseID: sourceClosed.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalClosed) error = %v", err)
	}
	linkedClosed, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-needs-follow-up",
		Title:              "Closed linked case",
		SourceEvalCaseID:   evalClosed.ID,
		SourceEvalReportID: "eval-report-closed",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed linked case) error = %v", err)
	}
	if _, err := caseService.CloseCase(ctx, linkedClosed.ID, "operator-close"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalService,
	}))
	defer server.Close()

	openResp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-needs-follow-up&needs_follow_up=true&limit=10")
	if err != nil {
		t.Fatalf("Get(openResp) error = %v", err)
	}
	defer openResp.Body.Close()
	if openResp.StatusCode != http.StatusOK {
		t.Fatalf("openResp StatusCode = %d, want %d", openResp.StatusCode, http.StatusOK)
	}
	var openBody listEvalCasesResponse
	if err := json.NewDecoder(openResp.Body).Decode(&openBody); err != nil {
		t.Fatalf("Decode(openBody) error = %v", err)
	}
	if len(openBody.EvalCases) != 1 || openBody.EvalCases[0].EvalCaseID != evalOpen.ID {
		t.Fatalf("openBody.EvalCases = %#v, want only %q", openBody.EvalCases, evalOpen.ID)
	}

	clearResp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-needs-follow-up&needs_follow_up=false&limit=10")
	if err != nil {
		t.Fatalf("Get(clearResp) error = %v", err)
	}
	defer clearResp.Body.Close()
	if clearResp.StatusCode != http.StatusOK {
		t.Fatalf("clearResp StatusCode = %d, want %d", clearResp.StatusCode, http.StatusOK)
	}
	var clearBody listEvalCasesResponse
	if err := json.NewDecoder(clearResp.Body).Decode(&clearBody); err != nil {
		t.Fatalf("Decode(clearBody) error = %v", err)
	}
	if len(clearBody.EvalCases) != 1 || clearBody.EvalCases[0].EvalCaseID != evalClosed.ID {
		t.Fatalf("clearBody.EvalCases = %#v, want only %q", clearBody.EvalCases, evalClosed.ID)
	}
}

func TestListEvalCasesEndpointRejectsInvalidNeedsFollowUp(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-invalid&needs_follow_up=maybe")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestEvalCaseEndpointsReturnFollowUpCaseSummary(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := evalsvc.NewService(caseService, nil)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:       "tenant-follow-up",
		Title:          "Source eval case",
		Summary:        "Promote this bad case",
		SourceTaskID:   "task-follow-up",
		SourceReportID: "report-follow-up",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	created, _, err := evalService.PromoteCase(ctx, evalsvc.CreateInput{
		TenantID:     "tenant-follow-up",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	followUp, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID:           "tenant-follow-up",
		Title:              "Follow-up case",
		Summary:            "Operator needs to inspect this bad case",
		SourceEvalReportID: "eval-report-follow-up",
		SourceEvalCaseID:   created.ID,
		CreatedBy:          "operator-follow-up",
	})
	if err != nil {
		t.Fatalf("CreateCase(followUp) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{
		Cases:     caseService,
		EvalCases: evalService,
	}))
	defer server.Close()

	listResp, err := http.Get(server.URL + "/api/v1/eval-cases?tenant_id=tenant-follow-up&limit=10")
	if err != nil {
		t.Fatalf("Get(list) error = %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list StatusCode = %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	var listBody listEvalCasesResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		t.Fatalf("Decode(list) error = %v", err)
	}
	if len(listBody.EvalCases) != 1 {
		t.Fatalf("len(EvalCases) = %d, want 1", len(listBody.EvalCases))
	}
	if listBody.EvalCases[0].FollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseCount = %d, want %d", listBody.EvalCases[0].FollowUpCaseCount, 1)
	}
	if listBody.EvalCases[0].OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want %d", listBody.EvalCases[0].OpenFollowUpCaseCount, 1)
	}
	if listBody.EvalCases[0].LatestFollowUpCaseID != followUp.ID {
		t.Fatalf("LatestFollowUpCaseID = %q, want %q", listBody.EvalCases[0].LatestFollowUpCaseID, followUp.ID)
	}
	if listBody.EvalCases[0].LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", listBody.EvalCases[0].LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}
	if listBody.EvalCases[0].PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("PreferredFollowUpAction.Mode = %q, want %q", listBody.EvalCases[0].PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if listBody.EvalCases[0].PreferredFollowUpAction.CaseID != followUp.ID {
		t.Fatalf("PreferredFollowUpAction.CaseID = %q, want %q", listBody.EvalCases[0].PreferredFollowUpAction.CaseID, followUp.ID)
	}

	detailResp, err := http.Get(server.URL + "/api/v1/eval-cases/" + created.ID + "?tenant_id=tenant-follow-up")
	if err != nil {
		t.Fatalf("Get(detail) error = %v", err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusOK {
		t.Fatalf("detail StatusCode = %d, want %d", detailResp.StatusCode, http.StatusOK)
	}

	var detail evalCaseResponse
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatalf("Decode(detail) error = %v", err)
	}
	if detail.FollowUpCaseCount != 1 {
		t.Fatalf("detail.FollowUpCaseCount = %d, want %d", detail.FollowUpCaseCount, 1)
	}
	if detail.LatestFollowUpCaseID != followUp.ID {
		t.Fatalf("detail.LatestFollowUpCaseID = %q, want %q", detail.LatestFollowUpCaseID, followUp.ID)
	}
	if detail.PreferredFollowUpAction.Mode != "open_existing_case" {
		t.Fatalf("detail.PreferredFollowUpAction.Mode = %q, want %q", detail.PreferredFollowUpAction.Mode, "open_existing_case")
	}
	if detail.PreferredFollowUpAction.CaseID != followUp.ID {
		t.Fatalf("detail.PreferredFollowUpAction.CaseID = %q, want %q", detail.PreferredFollowUpAction.CaseID, followUp.ID)
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

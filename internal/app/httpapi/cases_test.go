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

func (s *staleAssignStore) Assign(_ context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (casesvc.Case, error) {
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

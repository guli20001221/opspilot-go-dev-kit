package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"opspilot-go/internal/workflow"
)

func TestCreateTaskEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"report_generation","reason":"workflow_required"}`)
	resp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.TaskID == "" {
		t.Fatal("task_id is empty")
	}
	if got.Status != "queued" {
		t.Fatalf("status = %q, want %q", got.Status, "queued")
	}
}

func TestListTasksEndpointSupportsFiltersAndLimit(t *testing.T) {
	workflowService := workflow.NewService()

	task1, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-list-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(task1) error = %v", err)
	}
	task2, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID:        "req-list-2",
		TenantID:         "tenant-1",
		SessionID:        "session-2",
		TaskType:         workflow.TaskTypeApprovedToolExecution,
		Reason:           workflow.PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote(task2) error = %v", err)
	}
	task3, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-list-3",
		TenantID:  "tenant-2",
		SessionID: "session-3",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(task3) error = %v", err)
	}

	task1.Status = workflow.StatusSucceeded
	if _, err := workflowService.UpdateTask(context.Background(), task1); err != nil {
		t.Fatalf("UpdateTask(task1) error = %v", err)
	}
	task2.Status = workflow.StatusFailed
	task2.ErrorReason = "boom"
	if _, err := workflowService.UpdateTask(context.Background(), task2); err != nil {
		t.Fatalf("UpdateTask(task2) error = %v", err)
	}
	task3.Status = workflow.StatusFailed
	task3.ErrorReason = "boom"
	if _, err := workflowService.UpdateTask(context.Background(), task3); err != nil {
		t.Fatalf("UpdateTask(task3) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks?tenant_id=tenant-1&status=failed&task_type=approved_tool_execution&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got struct {
		Tasks      []taskResponse `json:"tasks"`
		HasMore    bool           `json:"has_more"`
		NextOffset *int           `json:"next_offset"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want %d", len(got.Tasks), 1)
	}
	if got.Tasks[0].TaskID != task2.ID {
		t.Fatalf("Tasks[0].TaskID = %q, want %q", got.Tasks[0].TaskID, task2.ID)
	}
	if len(got.Tasks[0].AuditEvents) != 0 {
		t.Fatalf("Tasks[0].AuditEvents = %#v, want omitted", got.Tasks[0].AuditEvents)
	}
	if got.HasMore {
		t.Fatal("HasMore = true, want false")
	}
	if got.NextOffset != nil {
		t.Fatalf("NextOffset = %v, want nil", *got.NextOffset)
	}
}

func TestListTasksEndpointSupportsOffsetPagination(t *testing.T) {
	workflowService := workflow.NewService()

	for i := 0; i < 3; i++ {
		created, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
			RequestID: "req-page",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  workflow.TaskTypeReportGeneration,
			Reason:    workflow.PromotionReasonWorkflowRequired,
		})
		if err != nil {
			t.Fatalf("Promote(%d) error = %v", i, err)
		}
		created.Status = workflow.StatusSucceeded
		if _, err := workflowService.UpdateTask(context.Background(), created); err != nil {
			t.Fatalf("UpdateTask(%d) error = %v", i, err)
		}
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	firstResp, err := http.Get(server.URL + "/api/v1/tasks?tenant_id=tenant-page&limit=1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer firstResp.Body.Close()

	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("firstResp.StatusCode = %d, want %d", firstResp.StatusCode, http.StatusOK)
	}

	var firstPage struct {
		Tasks      []taskResponse `json:"tasks"`
		HasMore    bool           `json:"has_more"`
		NextOffset *int           `json:"next_offset"`
	}
	if err := json.NewDecoder(firstResp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(firstPage.Tasks) != 1 {
		t.Fatalf("len(firstPage.Tasks) = %d, want %d", len(firstPage.Tasks), 1)
	}
	if !firstPage.HasMore {
		t.Fatal("firstPage.HasMore = false, want true")
	}
	if firstPage.NextOffset == nil || *firstPage.NextOffset != 1 {
		if firstPage.NextOffset == nil {
			t.Fatal("firstPage.NextOffset = nil, want 1")
		}
		t.Fatalf("firstPage.NextOffset = %d, want %d", *firstPage.NextOffset, 1)
	}

	secondResp, err := http.Get(server.URL + "/api/v1/tasks?tenant_id=tenant-page&limit=1&offset=1")
	if err != nil {
		t.Fatalf("Get(offset=1) error = %v", err)
	}
	defer secondResp.Body.Close()

	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("secondResp.StatusCode = %d, want %d", secondResp.StatusCode, http.StatusOK)
	}

	var secondPage struct {
		Tasks      []taskResponse `json:"tasks"`
		HasMore    bool           `json:"has_more"`
		NextOffset *int           `json:"next_offset"`
	}
	if err := json.NewDecoder(secondResp.Body).Decode(&secondPage); err != nil {
		t.Fatalf("Decode(secondPage) error = %v", err)
	}
	if len(secondPage.Tasks) != 1 {
		t.Fatalf("len(secondPage.Tasks) = %d, want %d", len(secondPage.Tasks), 1)
	}
	if secondPage.Tasks[0].TaskID == firstPage.Tasks[0].TaskID {
		t.Fatalf("secondPage.Tasks[0].TaskID = %q, want a different task from first page", secondPage.Tasks[0].TaskID)
	}
	if !secondPage.HasMore {
		t.Fatal("secondPage.HasMore = false, want true")
	}
	if secondPage.NextOffset == nil || *secondPage.NextOffset != 2 {
		if secondPage.NextOffset == nil {
			t.Fatal("secondPage.NextOffset = nil, want 2")
		}
		t.Fatalf("secondPage.NextOffset = %d, want %d", *secondPage.NextOffset, 2)
	}
}

func TestListTasksEndpointRejectsInvalidLimit(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks?limit=0")
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

func TestListTasksEndpointRejectsInvalidOffset(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks?offset=-1")
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

func TestGetTaskEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	createBody := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"report_generation","reason":"workflow_required"}`)
	createResp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", createBody)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer createResp.Body.Close()

	var created taskResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.TaskID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got struct {
		taskResponse
		AuditEvents []map[string]any `json:"audit_events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.TaskID != created.TaskID {
		t.Fatalf("task_id = %q, want %q", got.TaskID, created.TaskID)
	}
	if len(got.AuditEvents) == 0 {
		t.Fatal("audit_events is empty")
	}
}

func TestUnknownTaskReturnsJSONError(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "task_not_found" {
		t.Fatalf("code = %q, want %q", got.Code, "task_not_found")
	}
}

func TestApproveTaskEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	createBody := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"approved_tool_execution","reason":"approval_required","requires_approval":true}`)
	createResp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", createBody)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer createResp.Body.Close()

	var created taskResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	resp, err := http.Post(server.URL+"/api/v1/tasks/"+created.TaskID+"/approve", "application/json", bytes.NewBufferString(`{"approved_by":"operator-1"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Status != workflow.StatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, workflow.StatusQueued)
	}
}

func TestRetryTaskEndpoint(t *testing.T) {
	workflowService := workflow.NewService()
	created, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-retry",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	created.Status = workflow.StatusFailed
	created.ErrorReason = "boom"
	if _, err := workflowService.UpdateTask(context.Background(), created); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/tasks/"+created.ID+"/retry", "application/json", bytes.NewBufferString(`{"retried_by":"operator-2"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Status != workflow.StatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, workflow.StatusQueued)
	}
	if got.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", got.ErrorReason)
	}
}

func TestApproveTaskRejectsWrongState(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	createBody := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"report_generation","reason":"workflow_required"}`)
	createResp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", createBody)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer createResp.Body.Close()

	var created taskResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	resp, err := http.Post(server.URL+"/api/v1/tasks/"+created.TaskID+"/approve", "application/json", bytes.NewBufferString(`{"approved_by":"operator-1"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestGetTaskReturnsSummarizedFailureReason(t *testing.T) {
	workflowService := workflow.NewService()
	created, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-failure-summary",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeApprovedToolExecution,
		Reason:    workflow.PromotionReasonApprovalRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	created.Status = workflow.StatusFailed
	created.ErrorReason = "fault injection: approved tool failed on approve for task-1"
	if _, err := workflowService.UpdateTask(context.Background(), created); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.ErrorReason != "fault injection: approved tool failed on approve for task-1" {
		t.Fatalf("ErrorReason = %q, want summarized failure reason", got.ErrorReason)
	}
}

func TestGetTaskReturnsSucceededAuditSummary(t *testing.T) {
	workflowService := workflow.NewService()
	runner := workflow.NewRunner(workflowService, &fakeHTTPTaskExecutor{
		result: workflow.ExecutionResult{
			AuditRef: "worker:summary",
			Detail:   "ticket_comment_create comment_created for INC-222",
		},
	})

	created, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-success-summary",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.AuditEvents) == 0 {
		t.Fatal("AuditEvents is empty")
	}
	last := got.AuditEvents[len(got.AuditEvents)-1]
	if last.Action != workflow.AuditActionSucceeded {
		t.Fatalf("last.Action = %q, want %q", last.Action, workflow.AuditActionSucceeded)
	}
	if last.Detail != "ticket_comment_create comment_created for INC-222" {
		t.Fatalf("last.Detail = %q, want execution summary", last.Detail)
	}
}

func TestGetTaskReturnsCategorizedFailureAuditDetail(t *testing.T) {
	workflowService := workflow.NewService()
	runner := workflow.NewRunner(workflowService, &fakeHTTPTaskExecutor{
		err: errors.New("execute ticket_comment_create: ticket_comment_create requires ticket_id"),
		result: workflow.ExecutionResult{
			AuditRef: "worker:validation_failed",
		},
	})

	created, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-failure-detail",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeApprovedToolExecution,
		Reason:    workflow.PromotionReasonApprovalRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	last := got.AuditEvents[len(got.AuditEvents)-1]
	if last.Action != workflow.AuditActionFailed {
		t.Fatalf("last.Action = %q, want %q", last.Action, workflow.AuditActionFailed)
	}
	if last.Detail != "validation_error: execute ticket_comment_create: ticket_comment_create requires ticket_id" {
		t.Fatalf("last.Detail = %q, want categorized failure detail", last.Detail)
	}
}

type fakeHTTPTaskExecutor struct {
	result workflow.ExecutionResult
	err    error
}

func (f *fakeHTTPTaskExecutor) Execute(_ context.Context, _ workflow.Task) (workflow.ExecutionResult, error) {
	return f.result, f.err
}

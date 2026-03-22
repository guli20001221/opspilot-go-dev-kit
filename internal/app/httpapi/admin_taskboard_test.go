package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opspilot-go/internal/workflow"
)

func TestAdminTaskBoardEndpointReturnsBoardSummary(t *testing.T) {
	workflowService := workflow.NewService()

	first, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-admin-board-1",
		TenantID:  "tenant-admin-board",
		SessionID: "session-admin-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(first) error = %v", err)
	}
	second, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID:        "req-admin-board-2",
		TenantID:         "tenant-admin-board",
		SessionID:        "session-admin-2",
		TaskType:         workflow.TaskTypeApprovedToolExecution,
		Reason:           workflow.PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote(second) error = %v", err)
	}

	first.Status = workflow.StatusSucceeded
	if _, err := workflowService.UpdateTask(context.Background(), first); err != nil {
		t.Fatalf("UpdateTask(first) error = %v", err)
	}
	second.Status = workflow.StatusFailed
	second.ErrorReason = "approval failed"
	if _, err := workflowService.UpdateTask(context.Background(), second); err != nil {
		t.Fatalf("UpdateTask(second) error = %v", err)
	}

	server := httptest.NewServer(NewHandlerWithDependencies(Dependencies{Workflows: workflowService}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/admin/task-board?tenant_id=tenant-admin-board&limit=10")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got struct {
		Items []struct {
			TaskID      string `json:"task_id"`
			Status      string `json:"status"`
			AuditEvents []any  `json:"audit_events"`
		} `json:"items"`
		Page struct {
			HasMore    bool `json:"has_more"`
			NextOffset *int `json:"next_offset"`
		} `json:"page"`
		Summary struct {
			VisibleCount          int    `json:"visible_count"`
			RequiresApprovalCount int    `json:"requires_approval_count"`
			LatestFailureReason   string `json:"latest_failure_reason"`
			StatusCounts          struct {
				Succeeded int `json:"succeeded"`
				Failed    int `json:"failed"`
			} `json:"status_counts"`
			ReasonCounts struct {
				WorkflowRequired int `json:"workflow_required"`
				ApprovalRequired int `json:"approval_required"`
			} `json:"reason_counts"`
		} `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want %d", len(got.Items), 2)
	}
	if got.Summary.VisibleCount != 2 {
		t.Fatalf("Summary.VisibleCount = %d, want %d", got.Summary.VisibleCount, 2)
	}
	if got.Summary.RequiresApprovalCount != 1 {
		t.Fatalf("Summary.RequiresApprovalCount = %d, want %d", got.Summary.RequiresApprovalCount, 1)
	}
	if got.Summary.StatusCounts.Succeeded != 1 {
		t.Fatalf("Summary.StatusCounts.Succeeded = %d, want %d", got.Summary.StatusCounts.Succeeded, 1)
	}
	if got.Summary.StatusCounts.Failed != 1 {
		t.Fatalf("Summary.StatusCounts.Failed = %d, want %d", got.Summary.StatusCounts.Failed, 1)
	}
	if got.Summary.ReasonCounts.WorkflowRequired != 1 {
		t.Fatalf("Summary.ReasonCounts.WorkflowRequired = %d, want %d", got.Summary.ReasonCounts.WorkflowRequired, 1)
	}
	if got.Summary.ReasonCounts.ApprovalRequired != 1 {
		t.Fatalf("Summary.ReasonCounts.ApprovalRequired = %d, want %d", got.Summary.ReasonCounts.ApprovalRequired, 1)
	}
	if got.Summary.LatestFailureReason != "approval failed" {
		t.Fatalf("Summary.LatestFailureReason = %q, want %q", got.Summary.LatestFailureReason, "approval failed")
	}
	if got.Page.HasMore {
		t.Fatal("Page.HasMore = true, want false")
	}
	if got.Page.NextOffset != nil {
		t.Fatalf("Page.NextOffset = %v, want nil", *got.Page.NextOffset)
	}
	if len(got.Items[0].AuditEvents) != 0 {
		t.Fatalf("Items[0].AuditEvents = %#v, want omitted", got.Items[0].AuditEvents)
	}
}

func TestAdminTaskBoardEndpointRejectsInvalidQuery(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/admin/task-board?limit=0")
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

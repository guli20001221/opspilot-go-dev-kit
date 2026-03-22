package taskboard

import (
	"context"
	"testing"
	"time"

	"opspilot-go/internal/workflow"
)

func TestServiceListBuildsTaskBoardSummary(t *testing.T) {
	workflowService := workflow.NewService()

	first, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-board-1",
		TenantID:  "tenant-board",
		SessionID: "session-board-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(first) error = %v", err)
	}
	time.Sleep(time.Millisecond)
	second, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID:        "req-board-2",
		TenantID:         "tenant-board",
		SessionID:        "session-board-2",
		TaskType:         workflow.TaskTypeApprovedToolExecution,
		Reason:           workflow.PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote(second) error = %v", err)
	}
	time.Sleep(time.Millisecond)
	third, err := workflowService.Promote(context.Background(), workflow.PromoteRequest{
		RequestID: "req-board-3",
		TenantID:  "tenant-board",
		SessionID: "session-board-3",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote(third) error = %v", err)
	}

	first.Status = workflow.StatusSucceeded
	if _, err := workflowService.UpdateTask(context.Background(), first); err != nil {
		t.Fatalf("UpdateTask(first) error = %v", err)
	}
	second.Status = workflow.StatusFailed
	second.ErrorReason = "approval path failed"
	if _, err := workflowService.UpdateTask(context.Background(), second); err != nil {
		t.Fatalf("UpdateTask(second) error = %v", err)
	}
	third.Status = workflow.StatusRunning
	if _, err := workflowService.UpdateTask(context.Background(), third); err != nil {
		t.Fatalf("UpdateTask(third) error = %v", err)
	}

	service := NewService(workflowService)
	board, err := service.List(context.Background(), workflow.TaskListFilter{
		TenantID: "tenant-board",
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(board.Items) != 2 {
		t.Fatalf("len(board.Items) = %d, want %d", len(board.Items), 2)
	}
	if !board.Page.HasMore {
		t.Fatal("board.Page.HasMore = false, want true")
	}
	if board.Page.NextOffset == nil || *board.Page.NextOffset != 2 {
		if board.Page.NextOffset == nil {
			t.Fatal("board.Page.NextOffset = nil, want 2")
		}
		t.Fatalf("board.Page.NextOffset = %d, want %d", *board.Page.NextOffset, 2)
	}

	if board.Summary.VisibleCount != 2 {
		t.Fatalf("board.Summary.VisibleCount = %d, want %d", board.Summary.VisibleCount, 2)
	}
	if board.Summary.StatusCounts.Running != 1 {
		t.Fatalf("board.Summary.StatusCounts.Running = %d, want %d", board.Summary.StatusCounts.Running, 1)
	}
	if board.Summary.StatusCounts.Failed != 1 {
		t.Fatalf("board.Summary.StatusCounts.Failed = %d, want %d", board.Summary.StatusCounts.Failed, 1)
	}
	if board.Summary.RequiresApprovalCount != 1 {
		t.Fatalf("board.Summary.RequiresApprovalCount = %d, want %d", board.Summary.RequiresApprovalCount, 1)
	}
	if board.Summary.ReasonCounts.ApprovalRequired != 1 {
		t.Fatalf("board.Summary.ReasonCounts.ApprovalRequired = %d, want %d", board.Summary.ReasonCounts.ApprovalRequired, 1)
	}
	if board.Summary.ReasonCounts.WorkflowRequired != 1 {
		t.Fatalf("board.Summary.ReasonCounts.WorkflowRequired = %d, want %d", board.Summary.ReasonCounts.WorkflowRequired, 1)
	}
	if board.Summary.TaskTypeCounts.ApprovedToolExecution != 1 {
		t.Fatalf("board.Summary.TaskTypeCounts.ApprovedToolExecution = %d, want %d", board.Summary.TaskTypeCounts.ApprovedToolExecution, 1)
	}
	if board.Summary.TaskTypeCounts.ReportGeneration != 1 {
		t.Fatalf("board.Summary.TaskTypeCounts.ReportGeneration = %d, want %d", board.Summary.TaskTypeCounts.ReportGeneration, 1)
	}
	if board.Summary.LatestUpdatedAt == nil {
		t.Fatal("board.Summary.LatestUpdatedAt = nil, want value")
	}
	if board.Items[0].TaskID != third.ID {
		t.Fatalf("board.Items[0].TaskID = %q, want %q", board.Items[0].TaskID, third.ID)
	}
	if board.Items[1].TaskID != second.ID {
		t.Fatalf("board.Items[1].TaskID = %q, want %q", board.Items[1].TaskID, second.ID)
	}
}

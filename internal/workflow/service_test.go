package workflow

import (
	"context"
	"testing"
)

func TestServicePromoteCreatesQueuedTask(t *testing.T) {
	svc := NewService()

	got, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if got.ID == "" {
		t.Fatal("task ID is empty")
	}
	if got.Status != StatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, StatusQueued)
	}
	if got.Reason != PromotionReasonWorkflowRequired {
		t.Fatalf("Reason = %q, want %q", got.Reason, PromotionReasonWorkflowRequired)
	}
}

func TestServicePromoteCreatesWaitingApprovalTask(t *testing.T) {
	svc := NewService()

	got, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID:        "req-2",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         TaskTypeApprovedToolExecution,
		Reason:           PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if got.Status != StatusWaitingApproval {
		t.Fatalf("Status = %q, want %q", got.Status, StatusWaitingApproval)
	}
	if got.RequiresApproval != true {
		t.Fatal("RequiresApproval = false, want true")
	}
}

func TestServiceGetTaskReturnsStoredTask(t *testing.T) {
	svc := NewService()

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-3",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("ID = %q, want %q", got.ID, created.ID)
	}
}

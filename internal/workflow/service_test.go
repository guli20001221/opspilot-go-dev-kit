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

func TestServiceApproveTaskTransitionsWaitingApprovalToQueued(t *testing.T) {
	svc := NewService()

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID:        "req-4",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         TaskTypeApprovedToolExecution,
		Reason:           PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	got, err := svc.ApproveTask(context.Background(), created.ID, "operator-1")
	if err != nil {
		t.Fatalf("ApproveTask() error = %v", err)
	}
	if got.Status != StatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, StatusQueued)
	}
	if got.AuditRef != "approval:operator-1" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "approval:operator-1")
	}
}

func TestServiceRetryTaskTransitionsFailedToQueued(t *testing.T) {
	svc := NewService()

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-5",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	created.Status = StatusFailed
	created.ErrorReason = "boom"
	if _, err := svc.UpdateTask(context.Background(), created); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	got, err := svc.RetryTask(context.Background(), created.ID, "operator-2")
	if err != nil {
		t.Fatalf("RetryTask() error = %v", err)
	}
	if got.Status != StatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, StatusQueued)
	}
	if got.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", got.ErrorReason)
	}
	if got.AuditRef != "retry:operator-2" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "retry:operator-2")
	}
}

func TestServiceListTaskEventsReturnsStructuredHistory(t *testing.T) {
	svc := NewService()

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID:        "req-6",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         TaskTypeApprovedToolExecution,
		Reason:           PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if _, err := svc.ApproveTask(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("ApproveTask() error = %v", err)
	}

	events, err := svc.ListTaskEvents(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("len(events) = %d, want at least %d", len(events), 2)
	}
	if events[0].Action != AuditActionCreated {
		t.Fatalf("events[0].Action = %q, want %q", events[0].Action, AuditActionCreated)
	}
	if events[len(events)-1].Action != AuditActionApproved {
		t.Fatalf("events[last].Action = %q, want %q", events[len(events)-1].Action, AuditActionApproved)
	}
	if events[len(events)-1].Actor != "operator-1" {
		t.Fatalf("events[last].Actor = %q, want %q", events[len(events)-1].Actor, "operator-1")
	}
}

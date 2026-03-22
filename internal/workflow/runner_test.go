package workflow

import (
	"context"
	"testing"
)

func TestRunnerProcessesQueuedTaskToSucceeded(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, NewPlaceholderExecutor())

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	processed, err := runner.ProcessNextBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want %d", processed, 1)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}
	if got.AuditRef == "" {
		t.Fatal("AuditRef is empty")
	}

	events, err := svc.ListTaskEvents(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if events[len(events)-1].Action != AuditActionSucceeded {
		t.Fatalf("events[last].Action = %q, want %q", events[len(events)-1].Action, AuditActionSucceeded)
	}
}

func TestRunnerMarksUnsupportedTaskAsFailed(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, NewPlaceholderExecutor())

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-2",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  "unsupported_task_type",
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, StatusFailed)
	}
	if got.ErrorReason == "" {
		t.Fatal("ErrorReason is empty")
	}
}

func TestRunnerSkipsWaitingApprovalTasks(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, NewPlaceholderExecutor())

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID:        "req-3",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskType:         TaskTypeApprovedToolExecution,
		Reason:           PromotionReasonApprovalRequired,
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	processed, err := runner.ProcessNextBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 0 {
		t.Fatalf("processed = %d, want %d", processed, 0)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusWaitingApproval {
		t.Fatalf("Status = %q, want %q", got.Status, StatusWaitingApproval)
	}
}

func TestRunnerProcessesApprovedTaskAfterApproval(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, NewPlaceholderExecutor())

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

	if _, err := svc.ApproveTask(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("ApproveTask() error = %v", err)
	}
	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}
}

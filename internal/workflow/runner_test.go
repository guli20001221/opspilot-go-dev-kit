package workflow

import (
	"context"
	"errors"
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

func TestRunnerWritesExecutionSummaryIntoSucceededAuditEvent(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, &fakeRunnerExecutor{
		result: ExecutionResult{
			AuditRef: "worker:summary",
			Detail:   "ticket_comment_create comment_created for INC-222",
		},
	})

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-4b",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	events, err := svc.ListTaskEvents(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if events[len(events)-1].Action != AuditActionSucceeded {
		t.Fatalf("events[last].Action = %q, want %q", events[len(events)-1].Action, AuditActionSucceeded)
	}
	if events[len(events)-1].Detail != "ticket_comment_create comment_created for INC-222" {
		t.Fatalf("events[last].Detail = %q, want execution summary", events[len(events)-1].Detail)
	}
}

func TestRunnerSummarizesTemporalExecutionFailure(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, &fakeRunnerExecutor{
		err: errors.New("get approved tool workflow result: workflow execution error (type: ApprovedToolExecutionWorkflow, workflowID: task-1, runID: run-1): activity error (type: ExecuteApprovedTool, scheduledEventID: 9, startedEventID: 10, identity: worker): fault injection: approved tool failed on approve for task-1 (type: approved_tool_fault_injection, retryable: false)"),
		result: ExecutionResult{
			AuditRef: "temporal:workflow:task-1/run-1",
		},
	})

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
	if got.ErrorReason != "fault injection: approved tool failed on approve for task-1" {
		t.Fatalf("ErrorReason = %q, want summarized root cause", got.ErrorReason)
	}

	events, err := svc.ListTaskEvents(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if events[len(events)-1].Detail != "injected_failure: fault injection: approved tool failed on approve for task-1" {
		t.Fatalf("events[last].Detail = %q, want categorized root cause", events[len(events)-1].Detail)
	}
}

func TestRunnerClassifiesValidationFailureInAuditDetail(t *testing.T) {
	svc := NewService()
	runner := NewRunner(svc, &fakeRunnerExecutor{
		err: errors.New("execute ticket_comment_create: ticket_comment_create requires ticket_id"),
		result: ExecutionResult{
			AuditRef: "worker:validation_failed",
		},
	})

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-6",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeApprovedToolExecution,
		Reason:    PromotionReasonApprovalRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}

	events, err := svc.ListTaskEvents(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if events[len(events)-1].Detail != "validation_error: execute ticket_comment_create: ticket_comment_create requires ticket_id" {
		t.Fatalf("events[last].Detail = %q, want categorized validation summary", events[len(events)-1].Detail)
	}
}

func TestRunnerRecordsGeneratedReportOnSuccess(t *testing.T) {
	svc := NewService()
	recorder := &fakeReportRecorder{reportID: "report-task-1"}
	runner := NewRunnerWithReports(svc, &fakeRunnerExecutor{
		result: ExecutionResult{
			AuditRef: "temporal:workflow:task-1/run-1",
			Detail:   "generated:task-1",
		},
	}, recorder)

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-report-record",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
		Reason:    PromotionReasonWorkflowRequired,
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if _, err := runner.ProcessNextBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if recorder.calls != 1 {
		t.Fatalf("recorder.calls = %d, want %d", recorder.calls, 1)
	}
	if recorder.lastTaskID != created.ID {
		t.Fatalf("recorder.lastTaskID = %q, want %q", recorder.lastTaskID, created.ID)
	}
	if recorder.lastTaskAuditRef != "temporal:workflow:task-1/run-1" {
		t.Fatalf("recorder.lastTaskAuditRef = %q, want final success audit ref", recorder.lastTaskAuditRef)
	}
	if recorder.lastTaskStatus != StatusSucceeded {
		t.Fatalf("recorder.lastTaskStatus = %q, want %q", recorder.lastTaskStatus, StatusSucceeded)
	}
	if recorder.lastEvent.Action != AuditActionSucceeded {
		t.Fatalf("recorder.lastEvent.Action = %q, want %q", recorder.lastEvent.Action, AuditActionSucceeded)
	}
}

func TestRunnerFailsTaskWhenReportPersistenceFails(t *testing.T) {
	svc := NewService()
	runner := NewRunnerWithReports(svc, &fakeRunnerExecutor{
		result: ExecutionResult{
			AuditRef: "temporal:workflow:task-1/run-1",
			Detail:   "generated:task-1",
		},
	}, &fakeReportRecorder{err: errors.New("reports store unavailable")})

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-report-record-fail",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
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
	if got.ErrorReason != "reports store unavailable" {
		t.Fatalf("ErrorReason = %q, want %q", got.ErrorReason, "reports store unavailable")
	}
}

func TestRunnerFallsBackToNonAtomicReportRecorder(t *testing.T) {
	svc := NewService()
	recorder := &fakeNonAtomicReportRecorder{reportID: "report-task-non-atomic"}
	runner := NewRunnerWithReports(svc, &fakeRunnerExecutor{
		result: ExecutionResult{
			AuditRef: "temporal:workflow:task-1/run-1",
			Detail:   "generated:task-non-atomic",
		},
	}, recorder)

	created, err := svc.Promote(context.Background(), PromoteRequest{
		RequestID: "req-report-non-atomic",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  TaskTypeReportGeneration,
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
	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}
	if recorder.calls != 1 {
		t.Fatalf("recorder.calls = %d, want %d", recorder.calls, 1)
	}
	if recorder.lastTaskStatus != StatusSucceeded {
		t.Fatalf("recorder.lastTaskStatus = %q, want %q", recorder.lastTaskStatus, StatusSucceeded)
	}
	if recorder.lastTaskAuditRef != "temporal:workflow:task-1/run-1" {
		t.Fatalf("recorder.lastTaskAuditRef = %q, want final success audit ref", recorder.lastTaskAuditRef)
	}
}

type fakeRunnerExecutor struct {
	result ExecutionResult
	err    error
}

func (f *fakeRunnerExecutor) Execute(_ context.Context, _ Task) (ExecutionResult, error) {
	return f.result, f.err
}

type fakeReportRecorder struct {
	reportID         string
	err              error
	calls            int
	lastTaskID       string
	lastTaskAuditRef string
	lastTaskStatus   string
	lastEvent        AuditEvent
}

type fakeNonAtomicReportRecorder struct {
	reportID         string
	err              error
	calls            int
	lastTaskID       string
	lastTaskAuditRef string
	lastTaskStatus   string
}

func (f *fakeReportRecorder) RecordGeneratedReport(_ context.Context, task Task, _ ExecutionResult) (string, error) {
	f.calls++
	f.lastTaskID = task.ID
	f.lastTaskAuditRef = task.AuditRef
	f.lastTaskStatus = task.Status
	if f.err != nil {
		return "", f.err
	}

	return f.reportID, nil
}

func (f *fakeReportRecorder) FinalizeGeneratedReportTask(_ context.Context, task Task, _ ExecutionResult, event AuditEvent) (Task, string, error) {
	f.calls++
	f.lastTaskID = task.ID
	f.lastTaskAuditRef = task.AuditRef
	f.lastTaskStatus = task.Status
	f.lastEvent = event
	if f.err != nil {
		return Task{}, "", f.err
	}

	return task, f.reportID, nil
}

func (f *fakeReportRecorder) SupportsAtomicFinalization() bool {
	return true
}

func (f *fakeNonAtomicReportRecorder) RecordGeneratedReport(_ context.Context, task Task, _ ExecutionResult) (string, error) {
	f.calls++
	f.lastTaskID = task.ID
	f.lastTaskAuditRef = task.AuditRef
	f.lastTaskStatus = task.Status
	if f.err != nil {
		return "", f.err
	}

	return f.reportID, nil
}

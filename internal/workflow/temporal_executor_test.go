package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestTemporalExecutorUsesReportWorkflowRunnerForReportGeneration(t *testing.T) {
	reportRunner := &fakeReportWorkflowRunner{
		result: ExecutionResult{AuditRef: "temporal:report"},
	}
	fallback := &fakeExecutor{
		result: ExecutionResult{AuditRef: "fallback:report"},
	}
	executor := NewTemporalExecutor(reportRunner, &fakeApprovedToolWorkflowRunner{}, fallback)

	got, err := executor.Execute(context.Background(), Task{
		ID:       "task-report",
		TaskType: TaskTypeReportGeneration,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.AuditRef != "temporal:report" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "temporal:report")
	}
	if reportRunner.calls != 1 {
		t.Fatalf("reportRunner.calls = %d, want %d", reportRunner.calls, 1)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback.calls = %d, want %d", fallback.calls, 0)
	}
}

func TestTemporalExecutorFallsBackForNonReportTasks(t *testing.T) {
	reportRunner := &fakeReportWorkflowRunner{
		result: ExecutionResult{AuditRef: "temporal:report"},
	}
	approvalRunner := &fakeApprovedToolWorkflowRunner{
		result: ExecutionResult{AuditRef: "temporal:approved-tool"},
	}
	fallback := &fakeExecutor{
		result: ExecutionResult{AuditRef: "fallback:approved-tool"},
	}
	executor := NewTemporalExecutor(reportRunner, approvalRunner, fallback)

	got, err := executor.Execute(context.Background(), Task{
		ID:       "task-approved",
		TaskType: TaskTypeApprovedToolExecution,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.AuditRef != "fallback:approved-tool" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "fallback:approved-tool")
	}
	if reportRunner.calls != 0 {
		t.Fatalf("reportRunner.calls = %d, want %d", reportRunner.calls, 0)
	}
	if fallback.calls != 1 {
		t.Fatalf("fallback.calls = %d, want %d", fallback.calls, 1)
	}
	if approvalRunner.calls != 0 {
		t.Fatalf("approvalRunner.calls = %d, want %d", approvalRunner.calls, 0)
	}
}

func TestTemporalExecutorReturnsReportWorkflowError(t *testing.T) {
	reportRunner := &fakeReportWorkflowRunner{
		err: errors.New("temporal unavailable"),
	}
	executor := NewTemporalExecutor(reportRunner, &fakeApprovedToolWorkflowRunner{}, &fakeExecutor{})

	if _, err := executor.Execute(context.Background(), Task{
		ID:       "task-report-error",
		TaskType: TaskTypeReportGeneration,
	}); err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
}

func TestTemporalExecutorContinuesApprovedToolWorkflowAfterApproval(t *testing.T) {
	approvalRunner := &fakeApprovedToolWorkflowRunner{
		result: ExecutionResult{AuditRef: "temporal:approved-tool"},
	}
	fallback := &fakeExecutor{
		result: ExecutionResult{AuditRef: "fallback:approved-tool"},
	}
	executor := NewTemporalExecutor(&fakeReportWorkflowRunner{}, approvalRunner, fallback)

	got, err := executor.Execute(context.Background(), Task{
		ID:       "task-approved-after-approval",
		TaskType: TaskTypeApprovedToolExecution,
		AuditRef: "approval:operator-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.AuditRef != "temporal:approved-tool" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "temporal:approved-tool")
	}
	if approvalRunner.calls != 1 {
		t.Fatalf("approvalRunner.calls = %d, want %d", approvalRunner.calls, 1)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback.calls = %d, want %d", fallback.calls, 0)
	}
}

func TestTemporalExecutorContinuesApprovedToolWorkflowAfterRetry(t *testing.T) {
	approvalRunner := &fakeApprovedToolWorkflowRunner{
		result: ExecutionResult{AuditRef: "temporal:approved-tool-retry"},
	}
	executor := NewTemporalExecutor(&fakeReportWorkflowRunner{}, approvalRunner, &fakeExecutor{})

	got, err := executor.Execute(context.Background(), Task{
		ID:       "task-approved-after-retry",
		TaskType: TaskTypeApprovedToolExecution,
		AuditRef: "retry:operator-2",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.AuditRef != "temporal:approved-tool-retry" {
		t.Fatalf("AuditRef = %q, want %q", got.AuditRef, "temporal:approved-tool-retry")
	}
	if approvalRunner.calls != 1 {
		t.Fatalf("approvalRunner.calls = %d, want %d", approvalRunner.calls, 1)
	}
}

type fakeReportWorkflowRunner struct {
	result ExecutionResult
	err    error
	calls  int
}

func (f *fakeReportWorkflowRunner) RunReportWorkflow(_ context.Context, _ Task) (ExecutionResult, error) {
	f.calls++
	return f.result, f.err
}

type fakeApprovedToolWorkflowRunner struct {
	result ExecutionResult
	err    error
	calls  int
}

func (f *fakeApprovedToolWorkflowRunner) ContinueApprovedToolWorkflow(_ context.Context, _ Task) (ExecutionResult, error) {
	f.calls++
	return f.result, f.err
}

type fakeExecutor struct {
	result ExecutionResult
	err    error
	calls  int
}

func (f *fakeExecutor) Execute(_ context.Context, _ Task) (ExecutionResult, error) {
	f.calls++
	return f.result, f.err
}

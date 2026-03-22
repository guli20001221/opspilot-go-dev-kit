package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/stretchr/testify/mock"
)

func TestApprovedToolExecutionWorkflowReturnsActivityError(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	input := ApprovedToolWorkflowInput{
		TaskID:    "task-approved-error",
		TenantID:  "tenant-1",
		SessionID: "session-1",
	}

	env.OnActivity(new(ApprovedToolActivities).ExecuteApprovedTool, mock.Anything, ApprovedToolActivityInput{
		Workflow: input,
		Signal: ApprovedToolSignal{
			Action: "approve",
			Actor:  "operator-1",
		},
	}).
		Return(ApprovedToolWorkflowResult{}, temporal.NewNonRetryableApplicationError("boom", "approved_tool_failed", nil))
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(approvedToolContinueSignalName, ApprovedToolSignal{
			Action: "approve",
			Actor:  "operator-1",
		})
	}, 0)
	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(ApprovedToolExecutionWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("GetWorkflowError() = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("GetWorkflowError() = %v, want activity error containing %q", err, "boom")
	}
}

func TestApprovedToolStartWorkflowOptionsAllowDuplicateFailedOnlyForRetry(t *testing.T) {
	opts := approvedToolStartWorkflowOptions(Task{
		ID:       "task-approved-retry",
		AuditRef: "retry:operator-1",
	}, "opspilot-report-tasks")

	if opts.ID != "task-approved-retry" {
		t.Fatalf("ID = %q, want %q", opts.ID, "task-approved-retry")
	}
	if opts.TaskQueue != "opspilot-report-tasks" {
		t.Fatalf("TaskQueue = %q, want %q", opts.TaskQueue, "opspilot-report-tasks")
	}
	if opts.WorkflowIDReusePolicy != enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY {
		t.Fatalf("WorkflowIDReusePolicy = %v, want %v", opts.WorkflowIDReusePolicy, enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY)
	}
}

func TestApprovedToolActivitiesFailApproveWhenConfigured(t *testing.T) {
	activities := &ApprovedToolActivities{FailOnApprove: true}

	_, err := activities.ExecuteApprovedTool(context.Background(), ApprovedToolActivityInput{
		Workflow: ApprovedToolWorkflowInput{
			TaskID: "task-approved-fail-on-approve",
		},
		Signal: ApprovedToolSignal{
			Action: "approve",
			Actor:  "operator-1",
		},
	})
	if err == nil {
		t.Fatal("ExecuteApprovedTool() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "fault injection") {
		t.Fatalf("ExecuteApprovedTool() error = %v, want message containing %q", err, "fault injection")
	}
}

func TestApprovedToolActivitiesAllowRetryWhenConfigured(t *testing.T) {
	activities := &ApprovedToolActivities{FailOnApprove: true}

	got, err := activities.ExecuteApprovedTool(context.Background(), ApprovedToolActivityInput{
		Workflow: ApprovedToolWorkflowInput{
			TaskID: "task-approved-retry-pass",
		},
		Signal: ApprovedToolSignal{
			Action: "retry",
			Actor:  "operator-1",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedTool() error = %v", err)
	}
	if got.Executed != "approved-tool:task-approved-retry-pass" {
		t.Fatalf("Executed = %q, want %q", got.Executed, "approved-tool:task-approved-retry-pass")
	}
}

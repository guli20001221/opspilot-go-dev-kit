package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/stretchr/testify/mock"

	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
)

func TestApprovedToolExecutionWorkflowReturnsActivityError(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	input := ApprovedToolWorkflowInput{
		TaskID:    "task-approved-error",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		Arguments: json.RawMessage("null"),
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

func TestApprovedToolActivitiesExecuteApprovedToolFromPayload(t *testing.T) {
	activities := NewApprovedToolActivities(nil)

	got, err := activities.ExecuteApprovedTool(context.Background(), ApprovedToolActivityInput{
		Workflow: ApprovedToolWorkflowInput{
			TaskID:    "task-approved-runtime",
			TenantID:  "tenant-1",
			SessionID: "session-1",
			ToolName:  "ticket_comment_create",
			Arguments: []byte(`{"ticket_id":"INC-100","comment":"approved comment"}`),
		},
		Signal: ApprovedToolSignal{
			Action: "approve",
			Actor:  "operator-1",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedTool() error = %v", err)
	}
	if got.Executed != "ticket_comment_create comment_created for INC-100" {
		t.Fatalf("Executed = %q, want %q", got.Executed, "ticket_comment_create comment_created for INC-100")
	}
}

func TestApprovedToolActivitiesFallbackWithoutToolPayload(t *testing.T) {
	activities := NewApprovedToolActivities(nil)

	got, err := activities.ExecuteApprovedTool(context.Background(), ApprovedToolActivityInput{
		Workflow: ApprovedToolWorkflowInput{
			TaskID:    "task-approved-legacy",
			TenantID:  "tenant-1",
			SessionID: "session-1",
		},
		Signal: ApprovedToolSignal{
			Action: "approve",
			Actor:  "operator-1",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteApprovedTool() error = %v", err)
	}
	if got.Executed != "approved-tool:task-approved-legacy" {
		t.Fatalf("Executed = %q, want %q", got.Executed, "approved-tool:task-approved-legacy")
	}
}

func TestGenerateReportActivityWithSessionAndRetrieval(t *testing.T) {
	sessionService := newTestSessionService(t, "tenant-report", "user-report", "How do I reset my password?")
	activities := NewReportActivities(
		sessionService,
		newTestContextEngine(),
		newTestRetrievalService("tenant-report"),
	)

	result, err := activities.GenerateReport(context.Background(), ReportWorkflowInput{
		TaskID:    "task-report-1",
		TenantID:  "tenant-report",
		SessionID: sessionService.sessionID,
	})
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}
	if !strings.Contains(result.Generated, "Session: 1 messages loaded") {
		t.Fatalf("result missing session section: %q", result.Generated)
	}
	if !strings.Contains(result.Generated, "Context:") {
		t.Fatalf("result missing context section: %q", result.Generated)
	}
	if !strings.Contains(result.Generated, "Retrieval:") {
		t.Fatalf("result missing retrieval section: %q", result.Generated)
	}
	if !strings.Contains(result.Generated, "evidence blocks") {
		t.Fatalf("result missing evidence blocks: %q", result.Generated)
	}
}

func TestGenerateReportActivityGracefulDegradation(t *testing.T) {
	activities := NewReportActivities(nil, nil, nil)

	result, err := activities.GenerateReport(context.Background(), ReportWorkflowInput{
		TaskID:   "task-report-degraded",
		TenantID: "tenant-degraded",
	})
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}
	if !strings.Contains(result.Generated, "Session: unavailable") {
		t.Fatalf("result missing degraded session: %q", result.Generated)
	}
	if !strings.Contains(result.Generated, "Retrieval: skipped") {
		t.Fatalf("result missing skipped retrieval: %q", result.Generated)
	}
}

func TestGenerateReportActivityNoQueryText(t *testing.T) {
	sessionService := newTestSessionService(t, "tenant-noquery", "user-noquery", "")
	activities := NewReportActivities(
		sessionService,
		newTestContextEngine(),
		newTestRetrievalService("tenant-noquery"),
	)

	// Append an assistant message only (no user message → no query text)
	result, err := activities.GenerateReport(context.Background(), ReportWorkflowInput{
		TaskID:    "task-report-noquery",
		TenantID:  "tenant-noquery",
		SessionID: sessionService.sessionID,
	})
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}
	if !strings.Contains(result.Generated, "Retrieval: skipped") {
		t.Fatalf("expected retrieval skipped without query text: %q", result.Generated)
	}
}

// Test helpers

type testSessionService struct {
	sessionID string
	messages  []session.Message
}

func (s *testSessionService) ListMessages(_ context.Context, sessionID string) ([]session.Message, error) {
	if sessionID != s.sessionID {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	out := make([]session.Message, len(s.messages))
	copy(out, s.messages)
	return out, nil
}

func newTestSessionService(t *testing.T, tenantID, userID, userMessage string) *testSessionService {
	t.Helper()
	svc := &testSessionService{
		sessionID: fmt.Sprintf("sess-test-%d", time.Now().UnixNano()),
	}
	if userMessage != "" {
		svc.messages = []session.Message{
			{
				ID:        "msg-test-1",
				SessionID: svc.sessionID,
				Role:      session.RoleUser,
				Content:   userMessage,
				CreatedAt: time.Now().UTC(),
			},
		}
	}
	return svc
}

func newTestContextEngine() *contextengine.Service {
	return contextengine.NewService(contextengine.Config{})
}

func newTestRetrievalService(tenantID string) *retrieval.Service {
	return retrieval.NewService(nil)
}

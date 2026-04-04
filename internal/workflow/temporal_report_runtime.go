package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	temporalworker "go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"

	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
	toolregistry "opspilot-go/internal/tools/registry"
)

const (
	defaultReportWorkflowExecutionTimeout = 2 * time.Minute
	defaultReportWorkflowTaskTimeout      = 10 * time.Second
	defaultReportActivityTimeout          = 30 * time.Second
	approvedToolContinueSignalName        = "approved-tool-continue"
)

// TemporalOptions configures the Temporal client and worker task queue.
type TemporalOptions struct {
	Address   string
	Namespace string
	TaskQueue string
}

// ReportWorkflowInput carries the report task identity into Temporal.
type ReportWorkflowInput struct {
	TaskID    string
	TenantID  string
	SessionID string
}

// ReportWorkflowResult captures the placeholder report workflow outcome.
type ReportWorkflowResult struct {
	Generated string
}

// ApprovedToolWorkflowInput carries the approval-gated task identity into Temporal.
type ApprovedToolWorkflowInput struct {
	TaskID    string
	TenantID  string
	SessionID string
	ToolName  string
	Arguments json.RawMessage
}

// ApprovedToolActivityInput carries workflow identity plus the current action
// into the approved-tool activity.
type ApprovedToolActivityInput struct {
	Workflow ApprovedToolWorkflowInput
	Signal   ApprovedToolSignal
}

// ApprovedToolSignal carries the approval or retry signal actor into the workflow.
type ApprovedToolSignal struct {
	Action string
	Actor  string
}

// ApprovedToolWorkflowResult captures the placeholder approved-tool workflow outcome.
type ApprovedToolWorkflowResult struct {
	Executed string
}

// DialTemporalClient opens a Temporal client for the configured address and namespace.
func DialTemporalClient(opts TemporalOptions) (temporalclient.Client, error) {
	if opts.Address == "" {
		return nil, fmt.Errorf("temporal address must not be empty")
	}
	if opts.Namespace == "" {
		return nil, fmt.Errorf("temporal namespace must not be empty")
	}

	client, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  opts.Address,
		Namespace: opts.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("dial temporal client: %w", err)
	}

	return client, nil
}

// TemporalReportRunner executes report-generation tasks through Temporal.
type TemporalReportRunner struct {
	client     temporalclient.Client
	taskQueue  string
	activities *ReportActivities
}

// TemporalApprovedToolRunner manages approval-gated tool workflows in Temporal.
type TemporalApprovedToolRunner struct {
	client     temporalclient.Client
	taskQueue  string
	activities *ApprovedToolActivities
}

// NewTemporalReportRunner constructs a Temporal-backed report runner.
func NewTemporalReportRunner(client temporalclient.Client, taskQueue string) *TemporalReportRunner {
	return NewTemporalReportRunnerWithActivities(client, taskQueue, nil)
}

// NewTemporalReportRunnerWithActivities constructs a Temporal-backed report runner
// with caller-provided activity implementations.
func NewTemporalReportRunnerWithActivities(client temporalclient.Client, taskQueue string, activities *ReportActivities) *TemporalReportRunner {
	if activities == nil {
		activities = NewReportActivities(nil, nil, nil)
	}
	return &TemporalReportRunner{
		client:     client,
		taskQueue:  taskQueue,
		activities: activities,
	}
}

// NewTemporalApprovedToolRunner constructs a Temporal-backed approval workflow runner.
func NewTemporalApprovedToolRunner(client temporalclient.Client, taskQueue string) *TemporalApprovedToolRunner {
	return NewTemporalApprovedToolRunnerWithActivities(client, taskQueue, nil)
}

// NewTemporalApprovedToolRunnerWithActivities constructs a Temporal-backed
// approval workflow runner with caller-provided activity behavior.
func NewTemporalApprovedToolRunnerWithActivities(client temporalclient.Client, taskQueue string, activities *ApprovedToolActivities) *TemporalApprovedToolRunner {
	if activities == nil {
		activities = NewApprovedToolActivities(nil)
	}

	return &TemporalApprovedToolRunner{
		client:     client,
		taskQueue:  taskQueue,
		activities: activities,
	}
}

// RunReportWorkflow starts and waits for the report-generation workflow.
func (r *TemporalReportRunner) RunReportWorkflow(ctx context.Context, task Task) (ExecutionResult, error) {
	workflowRun, err := r.client.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:                       task.ID,
		TaskQueue:                r.taskQueue,
		WorkflowExecutionTimeout: defaultReportWorkflowExecutionTimeout,
		WorkflowTaskTimeout:      defaultReportWorkflowTaskTimeout,
	}, ReportGenerationWorkflow, ReportWorkflowInput{
		TaskID:    task.ID,
		TenantID:  task.TenantID,
		SessionID: task.SessionID,
	})
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("execute temporal report workflow: %w", err)
	}

	result := ExecutionResult{
		AuditRef: formatTemporalAuditRef(workflowRun.GetID(), workflowRun.GetRunID()),
	}

	var workflowResult ReportWorkflowResult
	if err := workflowRun.Get(ctx, &workflowResult); err != nil {
		return result, fmt.Errorf("get temporal report workflow result: %w", err)
	}
	result.Detail = workflowResult.Generated

	return result, nil
}

// Register registers the report workflow and its activities on a Temporal worker.
func (r *TemporalReportRunner) Register(w temporalworker.Worker) {
	w.RegisterWorkflow(ReportGenerationWorkflow)
	w.RegisterActivity(r.activities.GenerateReport)
}

// StartTask starts the waiting approval workflow on promote.
func (r *TemporalApprovedToolRunner) StartTask(ctx context.Context, task Task) error {
	_, err := r.client.ExecuteWorkflow(ctx, approvedToolStartWorkflowOptions(task, r.taskQueue), ApprovedToolExecutionWorkflow, ApprovedToolWorkflowInput{
		TaskID:    task.ID,
		TenantID:  task.TenantID,
		SessionID: task.SessionID,
		ToolName:  task.ToolName,
		Arguments: task.ToolArguments,
	})
	if err != nil {
		return fmt.Errorf("execute temporal approved tool workflow: %w", err)
	}

	return nil
}

// ContinueApprovedToolWorkflow signals the waiting approval workflow and waits
// for its completion.
func (r *TemporalApprovedToolRunner) ContinueApprovedToolWorkflow(ctx context.Context, task Task) (ExecutionResult, error) {
	signal := ApprovedToolSignal{
		Action: approvedToolSignalAction(task),
		Actor:  signalActorFromAuditRef(task.AuditRef),
	}
	workflowRun, err := r.client.SignalWithStartWorkflow(ctx,
		task.ID,
		approvedToolContinueSignalName,
		signal,
		approvedToolStartWorkflowOptions(task, r.taskQueue),
		ApprovedToolExecutionWorkflow,
		ApprovedToolWorkflowInput{
			TaskID:    task.ID,
			TenantID:  task.TenantID,
			SessionID: task.SessionID,
			ToolName:  task.ToolName,
			Arguments: task.ToolArguments,
		},
	)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("signal approved tool workflow: %w", err)
	}

	result := ExecutionResult{
		AuditRef: formatTemporalAuditRef(workflowRun.GetID(), workflowRun.GetRunID()),
	}

	var workflowResult ApprovedToolWorkflowResult
	if err := workflowRun.Get(ctx, &workflowResult); err != nil {
		return result, fmt.Errorf("get approved tool workflow result: %w", err)
	}
	result.Detail = workflowResult.Executed

	return result, nil
}

// Register registers the approval-gated workflow and its activities on a Temporal worker.
func (r *TemporalApprovedToolRunner) Register(w temporalworker.Worker) {
	w.RegisterWorkflow(ApprovedToolExecutionWorkflow)
	w.RegisterActivity(r.activities.ExecuteApprovedTool)
}

// NewTemporalWorker constructs a Temporal worker for the configured task queue.
func NewTemporalWorker(client temporalclient.Client, taskQueue string, reportRunner *TemporalReportRunner, approvedToolRunner *TemporalApprovedToolRunner) temporalworker.Worker {
	w := temporalworker.New(client, taskQueue, temporalworker.Options{})
	if reportRunner != nil {
		reportRunner.Register(w)
	}
	if approvedToolRunner != nil {
		approvedToolRunner.Register(w)
	}

	return w
}

// ReportGenerationWorkflow is the first Temporal-backed report workflow.
func ReportGenerationWorkflow(ctx temporalworkflow.Context, input ReportWorkflowInput) (ReportWorkflowResult, error) {
	ctx = temporalworkflow.WithActivityOptions(ctx, temporalworkflow.ActivityOptions{
		StartToCloseTimeout: defaultReportActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	var activities *ReportActivities
	var result ReportWorkflowResult
	if err := temporalworkflow.ExecuteActivity(ctx, activities.GenerateReport, input).Get(ctx, &result); err != nil {
		return ReportWorkflowResult{}, err
	}

	return result, nil
}

// ReportSessionReader is the narrow session interface consumed by report activities.
type ReportSessionReader interface {
	ListMessages(ctx context.Context, sessionID string) ([]session.Message, error)
}

// ReportActivities contains the activity implementations for report workflows.
type ReportActivities struct {
	sessions  ReportSessionReader
	contexts  *contextengine.Service
	retrieval *retrieval.Service
}

// NewReportActivities constructs report activities with caller-provided dependencies.
// Nil dependencies produce gracefully degraded reports.
func NewReportActivities(sessions ReportSessionReader, contexts *contextengine.Service, retrieval *retrieval.Service) *ReportActivities {
	return &ReportActivities{
		sessions:  sessions,
		contexts:  contexts,
		retrieval: retrieval,
	}
}

// GenerateReport assembles a report from session history and retrieval evidence.
func (a *ReportActivities) GenerateReport(ctx context.Context, input ReportWorkflowInput) (ReportWorkflowResult, error) {
	var sections []string

	// Load session messages
	var queryText string
	var sessionMessages []session.Message
	if a.sessions != nil && input.SessionID != "" {
		var err error
		sessionMessages, err = a.sessions.ListMessages(ctx, input.SessionID)
		if err != nil {
			return ReportWorkflowResult{}, fmt.Errorf("load session messages: %w", err)
		}
		for i := len(sessionMessages) - 1; i >= 0; i-- {
			if sessionMessages[i].Role == session.RoleUser {
				queryText = sessionMessages[i].Content
				break
			}
		}
		sections = append(sections, fmt.Sprintf("Session: %d messages loaded", len(sessionMessages)))
	} else {
		sections = append(sections, "Session: unavailable")
	}

	// Assemble context
	if a.contexts != nil {
		turns := make([]contextengine.Turn, 0, len(sessionMessages))
		for _, msg := range sessionMessages {
			turns = append(turns, contextengine.Turn{Role: msg.Role, Content: msg.Content})
		}
		assembled, err := a.contexts.Build(ctx, contextengine.BuildInput{
			RequestID:   input.TaskID,
			SessionID:   input.SessionID,
			TenantID:    input.TenantID,
			RecentTurns: turns,
		})
		if err != nil {
			return ReportWorkflowResult{}, fmt.Errorf("assemble context: %w", err)
		}
		sections = append(sections, fmt.Sprintf("Context: %d blocks assembled", len(assembled.Planner.Blocks)))
	}

	// Run retrieval
	if a.retrieval != nil && queryText != "" {
		result, err := a.retrieval.Search(ctx, retrieval.RetrievalRequest{
			RequestID: input.TaskID,
			TenantID:  input.TenantID,
			SessionID: input.SessionID,
			QueryText: queryText,
		})
		if err != nil {
			return ReportWorkflowResult{}, fmt.Errorf("retrieval search: %w", err)
		}
		sections = append(sections, fmt.Sprintf("Retrieval: %d evidence blocks, coverage %.2f", len(result.EvidenceBlocks), result.CoverageScore))
		for _, block := range result.EvidenceBlocks {
			sections = append(sections, fmt.Sprintf("  [%s] %s (score=%.2f)", block.CitationLabel, block.SourceTitle, block.Score))
		}
	} else {
		sections = append(sections, "Retrieval: skipped (no query text)")
	}

	return ReportWorkflowResult{
		Generated: strings.Join(sections, "\n"),
	}, nil
}

// ApprovedToolExecutionWorkflow is the first approval-gated Temporal workflow.
func ApprovedToolExecutionWorkflow(ctx temporalworkflow.Context, input ApprovedToolWorkflowInput) (ApprovedToolWorkflowResult, error) {
	ctx = temporalworkflow.WithActivityOptions(ctx, temporalworkflow.ActivityOptions{
		StartToCloseTimeout: defaultReportActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	signalChannel := temporalworkflow.GetSignalChannel(ctx, approvedToolContinueSignalName)
	var signal ApprovedToolSignal
	signalChannel.Receive(ctx, &signal)

	var activities *ApprovedToolActivities
	var result ApprovedToolWorkflowResult
	if err := temporalworkflow.ExecuteActivity(ctx, activities.ExecuteApprovedTool, ApprovedToolActivityInput{
		Workflow: input,
		Signal:   signal,
	}).Get(ctx, &result); err != nil {
		return ApprovedToolWorkflowResult{}, err
	}

	return result, nil
}

// ApprovedToolActivities contains the activity implementations for approved-tool workflows.
type ApprovedToolActivities struct {
	FailOnApprove bool
	tools         *agenttool.Service
}

// NewApprovedToolActivities constructs the approved-tool activity handler.
func NewApprovedToolActivities(tools *agenttool.Service) *ApprovedToolActivities {
	if tools == nil {
		tools = agenttool.NewService(toolregistry.NewDefaultRegistry())
	}

	return &ApprovedToolActivities{tools: tools}
}

// ExecuteApprovedTool is the placeholder approved-tool activity.
func (a *ApprovedToolActivities) ExecuteApprovedTool(_ context.Context, input ApprovedToolActivityInput) (ApprovedToolWorkflowResult, error) {
	if a.FailOnApprove && input.Signal.Action == "approve" {
		return ApprovedToolWorkflowResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("fault injection: approved tool failed on %s for %s", input.Signal.Action, input.Workflow.TaskID),
			"approved_tool_fault_injection",
			nil,
		)
	}
	if input.Workflow.ToolName == "" {
		return ApprovedToolWorkflowResult{
			Executed: fmt.Sprintf("approved-tool:%s", input.Workflow.TaskID),
		}, nil
	}
	if a.tools == nil {
		a.tools = agenttool.NewService(toolregistry.NewDefaultRegistry())
	}

	result, err := a.tools.Execute(context.Background(), agenttool.ToolInvocation{
		TenantID:         input.Workflow.TenantID,
		SessionID:        input.Workflow.SessionID,
		TaskID:           input.Workflow.TaskID,
		PlanID:           "workflow-" + input.Workflow.TaskID,
		StepID:           "approved-tool",
		ToolName:         input.Workflow.ToolName,
		ActionClass:      agenttool.ActionClassWrite,
		RequiresApproval: true,
		ApprovalGranted:  true,
		Arguments:        input.Workflow.Arguments,
	})
	if err != nil {
		return ApprovedToolWorkflowResult{}, err
	}
	if result.Status != agenttool.StatusSucceeded {
		return ApprovedToolWorkflowResult{}, fmt.Errorf("approved tool execution returned %s", result.Status)
	}

	return ApprovedToolWorkflowResult{
		Executed: summarizeApprovedToolResult(input.Workflow.ToolName, result.StructuredData),
	}, nil
}

func summarizeApprovedToolResult(toolName string, structuredData json.RawMessage) string {
	switch toolName {
	case "ticket_comment_create":
		var payload struct {
			TicketID string `json:"ticket_id"`
			Status   string `json:"status"`
		}
		if err := json.Unmarshal(structuredData, &payload); err == nil && payload.TicketID != "" && payload.Status != "" {
			return fmt.Sprintf("%s %s for %s", toolName, payload.Status, strings.ToUpper(payload.TicketID))
		}
	case "ticket_search":
		var payload struct {
			Matches []struct {
				TicketID string `json:"ticket_id"`
			} `json:"matches"`
		}
		if err := json.Unmarshal(structuredData, &payload); err == nil {
			return fmt.Sprintf("%s returned %d matches", toolName, len(payload.Matches))
		}
	}

	return toolName + " completed"
}

func formatTemporalAuditRef(workflowID string, runID string) string {
	return fmt.Sprintf("temporal:workflow:%s/%s", workflowID, runID)
}

func approvedToolSignalAction(task Task) string {
	if strings.HasPrefix(task.AuditRef, "retry:") {
		return "retry"
	}

	return "approve"
}

func signalActorFromAuditRef(auditRef string) string {
	if idx := strings.IndexByte(auditRef, ':'); idx >= 0 && idx < len(auditRef)-1 {
		return auditRef[idx+1:]
	}

	return ""
}

func approvedToolStartWorkflowOptions(task Task, taskQueue string) temporalclient.StartWorkflowOptions {
	opts := temporalclient.StartWorkflowOptions{
		ID:                       task.ID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: defaultReportWorkflowExecutionTimeout,
		WorkflowTaskTimeout:      defaultReportWorkflowTaskTimeout,
	}
	if strings.HasPrefix(task.AuditRef, "retry:") {
		opts.WorkflowIDReusePolicy = enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY
	}

	return opts
}

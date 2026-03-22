package workflow

import (
	"context"
	"fmt"
	"time"

	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	temporalworker "go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

const (
	defaultReportWorkflowExecutionTimeout = 2 * time.Minute
	defaultReportWorkflowTaskTimeout      = 10 * time.Second
	defaultReportActivityTimeout          = 30 * time.Second
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
	client    temporalclient.Client
	taskQueue string
}

// NewTemporalReportRunner constructs a Temporal-backed report runner.
func NewTemporalReportRunner(client temporalclient.Client, taskQueue string) *TemporalReportRunner {
	return &TemporalReportRunner{
		client:    client,
		taskQueue: taskQueue,
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

	return result, nil
}

// Register registers the report workflow and its activities on a Temporal worker.
func (r *TemporalReportRunner) Register(w temporalworker.Worker) {
	w.RegisterWorkflow(ReportGenerationWorkflow)
	w.RegisterActivity((&ReportActivities{}).GenerateReport)
}

// NewTemporalWorker constructs a Temporal worker for the configured task queue.
func NewTemporalWorker(client temporalclient.Client, taskQueue string, reportRunner *TemporalReportRunner) temporalworker.Worker {
	w := temporalworker.New(client, taskQueue, temporalworker.Options{})
	if reportRunner != nil {
		reportRunner.Register(w)
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

// ReportActivities contains the activity implementations for report workflows.
type ReportActivities struct{}

// GenerateReport is the placeholder report-generation activity.
func (a *ReportActivities) GenerateReport(_ context.Context, input ReportWorkflowInput) (ReportWorkflowResult, error) {
	return ReportWorkflowResult{
		Generated: fmt.Sprintf("generated:%s", input.TaskID),
	}, nil
}

func formatTemporalAuditRef(workflowID string, runID string) string {
	return fmt.Sprintf("temporal:workflow:%s/%s", workflowID, runID)
}

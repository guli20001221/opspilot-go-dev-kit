package workflow

import (
	"encoding/json"
	"time"
)

const (
	// StatusDraft identifies a created but not yet queued task.
	StatusDraft = "draft"
	// StatusQueued identifies a task accepted into the workflow layer.
	StatusQueued = "queued"
	// StatusRunning identifies a task currently being processed by a worker.
	StatusRunning = "running"
	// StatusSucceeded identifies a task completed successfully.
	StatusSucceeded = "succeeded"
	// StatusFailed identifies a task completed with failure details.
	StatusFailed = "failed"
	// StatusWaitingApproval identifies a task paused for manual approval.
	StatusWaitingApproval = "waiting_approval"

	// TaskTypeReportGeneration identifies report generation jobs.
	TaskTypeReportGeneration = "report_generation"
	// TaskTypeApprovedToolExecution identifies approval-gated tool execution jobs.
	TaskTypeApprovedToolExecution = "approved_tool_execution"

	// PromotionReasonWorkflowRequired identifies planner-driven promotion.
	PromotionReasonWorkflowRequired = "workflow_required"
	// PromotionReasonApprovalRequired identifies approval-driven promotion.
	PromotionReasonApprovalRequired = "approval_required"

	// AuditActionCreated identifies task creation events.
	AuditActionCreated = "created"
	// AuditActionClaimed identifies worker claim events.
	AuditActionClaimed = "claimed"
	// AuditActionApproved identifies approval resume events.
	AuditActionApproved = "approved"
	// AuditActionRetried identifies retry request events.
	AuditActionRetried = "retried"
	// AuditActionSucceeded identifies worker success events.
	AuditActionSucceeded = "succeeded"
	// AuditActionFailed identifies worker failure events.
	AuditActionFailed = "failed"
)

// PromoteRequest is the typed async-promotion request.
type PromoteRequest struct {
	RequestID        string
	TenantID         string
	SessionID        string
	TaskType         string
	Reason           string
	RequiresApproval bool
	ToolName         string
	ToolArguments    json.RawMessage
}

// Task is the typed async task record.
type Task struct {
	ID               string
	RequestID        string
	TenantID         string
	SessionID        string
	TaskType         string
	ToolName         string
	ToolArguments    json.RawMessage
	Status           string
	Reason           string
	ErrorReason      string
	AuditRef         string
	RequiresApproval bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AuditEvent is a structured task audit record.
type AuditEvent struct {
	ID        int64
	TaskID    string
	Action    string
	Actor     string
	Detail    string
	CreatedAt time.Time
}

// TaskListFilter narrows task list queries for operator-facing views.
type TaskListFilter struct {
	TenantID         string
	Status           string
	TaskType         string
	Reason           string
	RequiresApproval *bool
	Limit            int
	Offset           int
}

// TaskListPage is the paginated operator-facing task list result.
type TaskListPage struct {
	Tasks      []Task
	HasMore    bool
	NextOffset int
}

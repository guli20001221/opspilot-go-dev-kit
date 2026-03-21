package workflow

import "time"

const (
	// StatusDraft identifies a created but not yet queued task.
	StatusDraft = "draft"
	// StatusQueued identifies a task accepted into the workflow layer.
	StatusQueued = "queued"
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
)

// PromoteRequest is the typed async-promotion request.
type PromoteRequest struct {
	RequestID        string
	TenantID         string
	SessionID        string
	TaskType         string
	Reason           string
	RequiresApproval bool
}

// Task is the typed async task record.
type Task struct {
	ID               string
	RequestID        string
	TenantID         string
	SessionID        string
	TaskType         string
	Status           string
	Reason           string
	RequiresApproval bool
	CreatedAt        time.Time
}

package tool

import "encoding/json"

const (
	// ActionClassRead identifies read-only tools.
	ActionClassRead = "read"
	// ActionClassWrite identifies write-capable tools.
	ActionClassWrite = "write"
	// ActionClassAdmin identifies admin-only tools.
	ActionClassAdmin = "admin"

	// StatusSucceeded identifies successful tool execution.
	StatusSucceeded = "succeeded"
	// StatusFailed identifies failed tool execution.
	StatusFailed = "failed"
	// StatusApprovalRequired identifies approval-gated tool execution.
	StatusApprovalRequired = "approval_required"
)

// ToolInvocation is the typed tool request emitted by the planner/runtime.
type ToolInvocation struct {
	RequestID        string
	TraceID          string
	TenantID         string
	SessionID        string
	TaskID           string
	PlanID           string
	StepID           string
	ToolName         string
	ActionClass      string
	RequiresApproval bool
	Arguments        json.RawMessage
	DryRun           bool
}

// ToolResult is the normalized tool execution result.
type ToolResult struct {
	ToolCallID     string
	ToolName       string
	Status         string
	OutputSummary  string
	StructuredData json.RawMessage
	ErrorCode      string
	ErrorMessage   string
	ApprovalRef    string
	AuditRef       string
}

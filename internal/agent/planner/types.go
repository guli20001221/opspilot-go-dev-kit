package planner

import (
	"context"
	"encoding/json"

	"opspilot-go/internal/contextengine"
)

const (
	// IntentKnowledgeQA is the default intent for synchronous knowledge questions.
	IntentKnowledgeQA = "knowledge_qa"
	// IntentIncidentAssist identifies incident-support requests.
	IntentIncidentAssist = "incident_assist"
	// IntentReportRequest identifies report-generation requests.
	IntentReportRequest = "report_request"

	// StepKindRetrieve identifies retrieval plan steps.
	StepKindRetrieve = "retrieve"
	// StepKindTool identifies tool execution plan steps.
	StepKindTool = "tool"
	// StepKindSynthesize identifies answer composition plan steps.
	StepKindSynthesize = "synthesize"
	// StepKindCritic identifies critic validation plan steps.
	StepKindCritic = "critic"
	// StepKindPromoteWorkflow identifies promotion to async workflow.
	StepKindPromoteWorkflow = "promote_workflow"
)

// PlanInput is the typed planner request consumed by the runtime.
type PlanInput struct {
	RequestID       string
	TraceID         string
	TenantID        string
	SessionID       string
	Mode            string
	UserMessage     string
	Context         contextengine.PlannerContext
	AvailableTools  []ToolDescriptor
	TenantPolicy    TenantPolicy
	UserPermissions []string
}

// ToolDescriptor describes one tool available to the planner.
type ToolDescriptor struct {
	Name             string
	Description      string
	ReadOnly         bool
	RequiresApproval bool
	AsyncOnly        bool
	Parameters       []ToolParameterDesc
}

// ToolParameterDesc describes one expected parameter for a tool.
type ToolParameterDesc struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

// TenantPolicy contains the planner-facing policy controls for a tenant.
// When Configured is false, the planner uses permissive defaults.
type TenantPolicy struct {
	Configured              bool     // true when policy was explicitly loaded
	AllowToolUse            bool     // global toggle: if false, no tool steps allowed
	AllowToolUseExplicit    bool     // true when AllowToolUse was explicitly set (distinguishes "set to false" from "not set")
	AllowedTools            []string // if non-empty, only these tools are permitted
	ForbiddenTools          []string // these tools are always blocked
	MaxSteps                int      // max plan steps; 0 = use system default (6). Values above the system default are silently capped.
	RequireApprovalForWrite bool     // if true, all write tool steps must carry approval
}

// PolicyScope identifies the requesting scope for hierarchical policy resolution.
type PolicyScope struct {
	OrgID    string // organization-level scope (broadest)
	TenantID string // tenant-level scope
	UserID   string // user-level scope (most specific)
}

// PolicyLoader loads tenant policy at request time.
// Implementations may read from a database, config file, or return defaults.
type PolicyLoader interface {
	LoadPolicy(ctx context.Context, scope PolicyScope) TenantPolicy
}

// DefaultPolicyLoader returns a fixed permissive policy for all scopes.
// Replace with a database-backed implementation for production use.
type DefaultPolicyLoader struct{}

// LoadPolicy returns the permissive default (Configured=false).
func (DefaultPolicyLoader) LoadPolicy(_ context.Context, _ PolicyScope) TenantPolicy {
	return TenantPolicy{}
}

// StaticPolicyLoader returns a fixed policy for all scopes.
// Useful for local dev and integration testing.
type StaticPolicyLoader struct {
	Policy TenantPolicy
}

// LoadPolicy returns the static policy.
func (s StaticPolicyLoader) LoadPolicy(_ context.Context, _ PolicyScope) TenantPolicy {
	return s.Policy
}

// ExecutedStep records the outcome of one plan step during execution.
// Used as input to dynamic replanning.
type ExecutedStep struct {
	StepID   string
	Kind     string
	ToolName string
	Status   string // "succeeded", "failed", "approval_required"
	Summary  string // human-readable outcome (e.g. tool output summary or error)
}

// ReplanInput provides context for dynamic replanning after partial execution.
type ReplanInput struct {
	OriginalPlan  ExecutionPlan
	ExecutedSteps []ExecutedStep
	Input         PlanInput // original planner input for context
	ReplanReason  string    // why replanning was triggered (e.g. "tool ticket_search failed: not found")
}

// PlanSourceReplan indicates the plan was produced by LLM dynamic replanning.
const PlanSourceReplan = "replan"

// PlanSourceKeyword indicates the plan was produced by keyword-based fallback.
const PlanSourceKeyword = "keyword"

// PlanSourceLLM indicates the plan was produced by LLM structured output.
const PlanSourceLLM = "llm"

// ExecutionPlan is the structured planner output for the current request.
type ExecutionPlan struct {
	PlanID                string
	Intent                string
	RequiresRetrieval     bool
	RequiresTool          bool
	RequiresWorkflow      bool
	RequiresApproval      bool
	MaxSteps              int
	OutputSchema          string
	Steps                 []PlanStep
	PlannerReasoningShort string
	Source                string // "keyword" or "llm" — indicates how the plan was produced
	PromptVersion         string // prompt version used for LLM plans (empty for keyword)
}

// PlanStep is one auditable planner action.
type PlanStep struct {
	StepID        string
	Kind          string
	Name          string
	DependsOn     []string
	ToolName      string
	ToolArguments json.RawMessage // planner-produced structured arguments; nil triggers heuristic fallback
	ReadOnly      bool
	NeedsApproval bool
}

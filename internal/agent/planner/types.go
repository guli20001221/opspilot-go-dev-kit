package planner

import "opspilot-go/internal/contextengine"

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
	ReadOnly         bool
	RequiresApproval bool
	AsyncOnly        bool
}

// TenantPolicy contains the minimal planner-facing policy toggles.
type TenantPolicy struct {
	AllowToolUse bool
}

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
}

// PlanStep is one auditable planner action.
type PlanStep struct {
	StepID        string
	Kind          string
	Name          string
	DependsOn     []string
	ToolName      string
	ReadOnly      bool
	NeedsApproval bool
}

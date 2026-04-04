package critic

import (
	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/retrieval"
)

const (
	// VerdictApprove indicates the draft is acceptable as-is.
	VerdictApprove = "approve"
	// VerdictRevise indicates the draft can be improved in the sync path.
	VerdictRevise = "revise"
	// VerdictPromoteWorkflow indicates the work should move to async workflow.
	VerdictPromoteWorkflow = "promote_workflow"
	// VerdictReject indicates the draft should be blocked.
	VerdictReject = "reject"

	// RiskLevelLow indicates low risk output.
	RiskLevelLow = "low"
	// RiskLevelMedium indicates medium risk output.
	RiskLevelMedium = "medium"
	// RiskLevelHigh indicates high risk output.
	RiskLevelHigh = "high"
)

// CriticInput is the typed review request.
type CriticInput struct {
	Plan        planner.ExecutionPlan
	Retrieval   *retrieval.RetrievalResult
	ToolResults []agenttool.ToolResult
	DraftAnswer string
}

const (
	// CriticSourceLLM indicates the verdict was produced by an LLM.
	CriticSourceLLM = "llm"
	// CriticSourceRule indicates the verdict was produced by deterministic rules.
	CriticSourceRule = "rule"
)

// CriticVerdict is the structured review result.
type CriticVerdict struct {
	Verdict          string
	Groundedness     float64
	CitationCoverage float64
	ToolConsistency  float64
	RiskLevel        string
	MissingItems     []string
	RevisionHints    []string
	BlockingReasons  []string
	Source           string
	PromptVersion    string
}

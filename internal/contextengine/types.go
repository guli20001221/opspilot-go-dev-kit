package contextengine

import "context"

const (
	// BlockKindUserProfile identifies the user and tenant scope block.
	BlockKindUserProfile = "user_profile"
	// BlockKindRecentTurns identifies the recent conversation turns block.
	BlockKindRecentTurns = "recent_turns"
	// BlockKindSessionSummary identifies the replaceable summary artifact block.
	BlockKindSessionSummary = "session_summary"
	// BlockKindTaskScratchpad identifies the current task notes block.
	BlockKindTaskScratchpad = "task_scratchpad"
	// BlockKindRetrievalEvidence identifies retrieved evidence snippets.
	BlockKindRetrievalEvidence = "retrieval_evidence"
	// BlockKindToolResult identifies tool execution output.
	BlockKindToolResult = "tool_result"
)

// Summarizer compresses older conversation turns into a concise summary.
// When the context engine detects that recent turns exceed a threshold,
// it calls the summarizer to compress the oldest turns into a summary block.
// This implements the ConversationSummaryBuffer pattern from LangChain.
type Summarizer interface {
	Summarize(ctx context.Context, turns []Turn) (string, error)
}

// Config controls deterministic block assembly limits.
// Per-stage budgets default to the global Budget when zero.
type Config struct {
	MaxBlocks       int
	Budget          int // global token budget (used as default for all stages)
	PlannerBudget   int // 0 = use Budget
	RetrievalBudget int // 0 = use Budget
	CriticBudget    int // 0 = use Budget
	// SummaryTurnThreshold: when recent turns exceed this count, older turns
	// are compressed into a session summary via the Summarizer. 0 = disabled.
	SummaryTurnThreshold int
}

// Turn is the minimal conversation unit used during context assembly.
type Turn struct {
	Role    string
	Content string
}

// EvidenceSnippet is one retrieval evidence item for context assembly.
type EvidenceSnippet struct {
	SourceTitle   string
	Snippet       string
	CitationLabel string
	Score         float64
}

// ToolResultSnippet is one tool execution output for context assembly.
type ToolResultSnippet struct {
	ToolName      string
	Status        string
	OutputSummary string
}

// BuildInput contains the current request metadata and available context sources.
type BuildInput struct {
	RequestID        string
	SessionID        string
	TenantID         string
	UserID           string
	Mode             string
	RecentTurns      []Turn
	SessionSummary   string
	TaskScratchpad   string
	RetrievalResults []EvidenceSnippet  // populated after retrieval phase
	ToolResults      []ToolResultSnippet // populated after tool execution phase
}

// Block is one included context block with deterministic metadata.
type Block struct {
	Kind            string
	Content         string
	IncludeReason   string
	EstimatedTokens int
	Priority        int
}

// PlannerContext is the planner-specific assembled context snapshot.
type PlannerContext struct {
	Blocks []Block
}

// RetrievalContext is the retrieval-specific assembled context snapshot.
type RetrievalContext struct {
	Blocks []Block
}

// CriticContext is the critic-specific assembled context snapshot.
type CriticContext struct {
	Blocks []Block
}

// AssemblyLog records which blocks were included or dropped during assembly.
// IncludedBlocks reflects the planner stage (most constrained).
// DroppedBlocks aggregates drops across all three stages.
// BudgetUsed/BudgetLimit reflect the critic stage (most permissive).
type AssemblyLog struct {
	RequestID      string
	IncludedBlocks []string
	DroppedBlocks  []string
	BudgetUsed     int
	BudgetLimit    int
}

// BuildResult contains the stage-specific context snapshots and the assembly log.
type BuildResult struct {
	Planner   PlannerContext
	Retrieval RetrievalContext
	Critic    CriticContext
	Log       AssemblyLog
}

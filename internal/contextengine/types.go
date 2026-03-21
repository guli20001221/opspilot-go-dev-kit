package contextengine

const (
	// BlockKindUserProfile identifies the user and tenant scope block.
	BlockKindUserProfile = "user_profile"
	// BlockKindRecentTurns identifies the recent conversation turns block.
	BlockKindRecentTurns = "recent_turns"
	// BlockKindSessionSummary identifies the replaceable summary artifact block.
	BlockKindSessionSummary = "session_summary"
	// BlockKindTaskScratchpad identifies the current task notes block.
	BlockKindTaskScratchpad = "task_scratchpad"
)

// Config controls deterministic block assembly limits.
type Config struct {
	MaxBlocks int
	Budget    int
}

// Turn is the minimal conversation unit used during context assembly.
type Turn struct {
	Role    string
	Content string
}

// BuildInput contains the current request metadata and available context sources.
type BuildInput struct {
	RequestID      string
	SessionID      string
	TenantID       string
	UserID         string
	Mode           string
	RecentTurns    []Turn
	SessionSummary string
	TaskScratchpad string
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

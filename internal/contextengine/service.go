package contextengine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const (
	defaultMaxBlocks = 8
	defaultBudget    = 128
)

// Service assembles explicit context blocks from request and session state.
type Service struct {
	config     Config
	summarizer Summarizer
}

// NewService constructs a context assembly service with deterministic defaults.
func NewService(config Config) *Service {
	return NewServiceWithSummarizer(config, nil)
}

// NewServiceWithSummarizer constructs a context assembly service with an optional
// conversation summarizer for compressing long turn histories.
func NewServiceWithSummarizer(config Config, summarizer Summarizer) *Service {
	if config.MaxBlocks <= 0 {
		config.MaxBlocks = defaultMaxBlocks
	}
	if config.Budget <= 0 {
		config.Budget = defaultBudget
	}

	return &Service{config: config, summarizer: summarizer}
}

// Build assembles stage-specific context snapshots for planner, retrieval, and critic.
// Each stage gets a different subset of blocks filtered by relevance and budget.
// When a Summarizer is configured and recent turns exceed SummaryTurnThreshold,
// older turns are compressed into a session summary (ConversationSummaryBuffer pattern).
func (s *Service) Build(ctx context.Context, input BuildInput) (BuildResult, error) {
	// ConversationSummaryBuffer: compress older turns when threshold is exceeded
	input = s.maybeSummarize(ctx, input)

	allBlocks := s.candidateBlocks(input)

	// Planner: needs user profile, recent turns, scratchpad (no evidence/tool results)
	plannerBlocks := filterByKinds(allBlocks,
		BlockKindUserProfile, BlockKindRecentTurns, BlockKindSessionSummary, BlockKindTaskScratchpad)
	plannerBlocks, plannerDropped := s.applyBudget(plannerBlocks, s.plannerBudget())

	// Retrieval: needs user profile, recent turns, summary (for query context)
	retrievalBlocks := filterByKinds(allBlocks,
		BlockKindUserProfile, BlockKindRecentTurns, BlockKindSessionSummary)
	retrievalBlocks, retrievalDropped := s.applyBudget(retrievalBlocks, s.retrievalBudget())

	// Critic: needs everything — evidence and tool results for validation
	criticBlocks, criticDropped := s.applyBudget(allBlocks, s.criticBudget())

	return BuildResult{
		Planner:   PlannerContext{Blocks: plannerBlocks},
		Retrieval: RetrievalContext{Blocks: retrievalBlocks},
		Critic:    CriticContext{Blocks: criticBlocks},
		Log: AssemblyLog{
			RequestID:      input.RequestID,
			IncludedBlocks: blockKinds(plannerBlocks),
			DroppedBlocks:  append(append(blockKinds(plannerDropped), blockKinds(retrievalDropped)...), blockKinds(criticDropped)...),
			BudgetUsed:     totalEstimatedTokens(criticBlocks), // report largest stage
			BudgetLimit:    s.criticBudget(),
		},
	}, nil
}

// maybeSummarize implements the ConversationSummaryBuffer pattern:
// when recent turns exceed the threshold and a summarizer is available,
// compress the oldest turns into a session summary while keeping the
// most recent turns as-is. This prevents token overflow from long conversations
// while preserving recent conversational context in full fidelity.
func (s *Service) maybeSummarize(ctx context.Context, input BuildInput) BuildInput {
	threshold := s.config.SummaryTurnThreshold
	if threshold <= 0 || s.summarizer == nil || len(input.RecentTurns) <= threshold {
		return input
	}

	// Keep the most recent `threshold` turns in full; compress the rest
	splitAt := len(input.RecentTurns) - threshold
	olderTurns := input.RecentTurns[:splitAt]
	recentTurns := input.RecentTurns[splitAt:]

	summary, err := s.summarizer.Summarize(ctx, olderTurns)
	if err != nil {
		slog.Warn("conversation summarization failed, using full turns",
			slog.String("request_id", input.RequestID),
			slog.Any("error", err),
		)
		return input
	}

	// Prepend the new summary to any existing session summary
	if input.SessionSummary != "" {
		input.SessionSummary = input.SessionSummary + "\n\n" + summary
	} else {
		input.SessionSummary = summary
	}
	input.RecentTurns = recentTurns

	slog.Debug("conversation summarized",
		slog.String("request_id", input.RequestID),
		slog.Int("compressed_turns", len(olderTurns)),
		slog.Int("kept_turns", len(recentTurns)),
	)

	return input
}

func (s *Service) candidateBlocks(input BuildInput) []Block {
	var blocks []Block

	if content := formatUserProfile(input); content != "" {
		blocks = append(blocks, newBlock(BlockKindUserProfile, content, "request tenant and user scope", 100))
	}
	if content := formatRecentTurns(input.RecentTurns); content != "" {
		blocks = append(blocks, newBlock(BlockKindRecentTurns, content, "recent session turns", 90))
	}
	if input.SessionSummary != "" {
		blocks = append(blocks, newBlock(BlockKindSessionSummary, input.SessionSummary, "replaceable session summary", 50))
	}
	if input.TaskScratchpad != "" {
		blocks = append(blocks, newBlock(BlockKindTaskScratchpad, input.TaskScratchpad, "active task notes", 40))
	}
	// Evidence snippets as individual blocks for fine-grained budget eviction.
	// Lower-scored evidence gets lower priority so it drops first under pressure.
	for i, ev := range input.RetrievalResults {
		label := ev.CitationLabel
		if label == "" {
			label = fmt.Sprintf("[%d]", i+1)
		}
		content := fmt.Sprintf("%s %s: %s (score=%.3f)", label, ev.SourceTitle, ev.Snippet, ev.Score)
		// Priority 80 base, adjusted by position (earlier = higher relevance after reranking)
		priority := 80 - i
		if priority < 60 {
			priority = 60
		}
		blocks = append(blocks, newBlock(BlockKindRetrievalEvidence, content,
			fmt.Sprintf("evidence snippet %s (score=%.3f)", label, ev.Score), priority))
	}
	// Tool results as individual blocks for per-tool eviction control
	for _, tr := range input.ToolResults {
		content := fmt.Sprintf("[%s] %s: %s", tr.Status, tr.ToolName, tr.OutputSummary)
		blocks = append(blocks, newBlock(BlockKindToolResult, content,
			fmt.Sprintf("tool output from %s (%s)", tr.ToolName, tr.Status), 70))
	}

	return blocks
}

func (s *Service) plannerBudget() int {
	if s.config.PlannerBudget > 0 {
		return s.config.PlannerBudget
	}
	return s.config.Budget
}

func (s *Service) retrievalBudget() int {
	if s.config.RetrievalBudget > 0 {
		return s.config.RetrievalBudget
	}
	return s.config.Budget
}

func (s *Service) criticBudget() int {
	if s.config.CriticBudget > 0 {
		return s.config.CriticBudget
	}
	return s.config.Budget
}

func (s *Service) applyBudget(blocks []Block, budget int) ([]Block, []Block) {
	included := cloneBlocks(blocks)
	var dropped []Block

	for len(included) > s.config.MaxBlocks {
		idx := lowestPriorityIndex(included)
		dropped = append(dropped, included[idx])
		included = removeBlock(included, idx)
	}

	for len(included) > 0 && totalEstimatedTokens(included) > budget {
		idx := lowestPriorityIndex(included)
		dropped = append(dropped, included[idx])
		included = removeBlock(included, idx)
	}

	// Safety net: never return empty context. Keep the highest-priority block
	// even if it exceeds budget — an over-budget context is better than no context.
	if len(included) == 0 && len(blocks) > 0 {
		return []Block{blocks[0]}, dropped
	}

	return included, dropped
}

func filterByKinds(blocks []Block, kinds ...string) []Block {
	kindSet := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	filtered := make([]Block, 0, len(blocks))
	for _, b := range blocks {
		if kindSet[b.Kind] {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func newBlock(kind string, content string, reason string, priority int) Block {
	return Block{
		Kind:            kind,
		Content:         content,
		IncludeReason:   reason,
		EstimatedTokens: estimateTokens(content),
		Priority:        priority,
	}
}

func formatUserProfile(input BuildInput) string {
	lines := make([]string, 0, 3)
	if input.TenantID != "" {
		lines = append(lines, fmt.Sprintf("tenant_id=%s", input.TenantID))
	}
	if input.UserID != "" {
		lines = append(lines, fmt.Sprintf("user_id=%s", input.UserID))
	}
	if input.Mode != "" {
		lines = append(lines, fmt.Sprintf("mode=%s", input.Mode))
	}

	return strings.Join(lines, "\n")
}

func formatRecentTurns(turns []Turn) string {
	lines := make([]string, 0, len(turns))
	for _, turn := range turns {
		if turn.Role == "" && turn.Content == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", turn.Role, turn.Content))
	}

	return strings.Join(lines, "\n")
}

func estimateTokens(content string) int {
	if content == "" {
		return 0
	}

	return (len(content) + 3) / 4
}

func totalEstimatedTokens(blocks []Block) int {
	total := 0
	for _, block := range blocks {
		total += block.EstimatedTokens
	}

	return total
}

func lowestPriorityIndex(blocks []Block) int {
	idx := 0
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Priority < blocks[idx].Priority {
			idx = i
		}
	}

	return idx
}

func removeBlock(blocks []Block, idx int) []Block {
	out := make([]Block, 0, len(blocks)-1)
	out = append(out, blocks[:idx]...)
	out = append(out, blocks[idx+1:]...)
	return out
}

func cloneBlocks(blocks []Block) []Block {
	out := make([]Block, len(blocks))
	copy(out, blocks)
	return out
}

func blockKinds(blocks []Block) []string {
	out := make([]string, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, block.Kind)
	}

	return out
}

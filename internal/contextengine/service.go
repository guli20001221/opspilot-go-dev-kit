package contextengine

import (
	"context"
	"fmt"
	"strings"
)

const (
	defaultMaxBlocks = 8
	defaultBudget    = 128
)

// Service assembles explicit context blocks from request and session state.
type Service struct {
	config Config
}

// NewService constructs a context assembly service with deterministic defaults.
func NewService(config Config) *Service {
	if config.MaxBlocks <= 0 {
		config.MaxBlocks = defaultMaxBlocks
	}
	if config.Budget <= 0 {
		config.Budget = defaultBudget
	}

	return &Service{config: config}
}

// Build assembles planner, retrieval, and critic contexts from the same block set.
func (s *Service) Build(_ context.Context, input BuildInput) (BuildResult, error) {
	blocks := s.candidateBlocks(input)
	blocks, dropped := s.applyLimits(blocks)

	return BuildResult{
		Planner:   PlannerContext{Blocks: cloneBlocks(blocks)},
		Retrieval: RetrievalContext{Blocks: cloneBlocks(blocks)},
		Critic:    CriticContext{Blocks: cloneBlocks(blocks)},
		Log: AssemblyLog{
			RequestID:      input.RequestID,
			IncludedBlocks: blockKinds(blocks),
			DroppedBlocks:  blockKinds(dropped),
			BudgetUsed:     totalEstimatedTokens(blocks),
			BudgetLimit:    s.config.Budget,
		},
	}, nil
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

	return blocks
}

func (s *Service) applyLimits(blocks []Block) ([]Block, []Block) {
	included := cloneBlocks(blocks)
	var dropped []Block

	for len(included) > s.config.MaxBlocks {
		idx := lowestPriorityIndex(included)
		dropped = append(dropped, included[idx])
		included = removeBlock(included, idx)
	}

	for len(included) > 0 && totalEstimatedTokens(included) > s.config.Budget {
		idx := lowestPriorityIndex(included)
		dropped = append(dropped, included[idx])
		included = removeBlock(included, idx)
	}

	if len(included) == 0 && len(blocks) > 0 {
		return []Block{blocks[0]}, dropped
	}

	return included, dropped
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

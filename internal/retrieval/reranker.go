package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"opspilot-go/internal/llm"
)

// Reranker re-scores evidence blocks for relevance to the query.
type Reranker interface {
	Rerank(ctx context.Context, query string, blocks []EvidenceBlock) ([]EvidenceBlock, error)
}

// LLMReranker uses a large language model to score (query, passage) relevance.
// Based on RankGPT-style pointwise scoring: for each passage, the LLM rates
// relevance on a 0-10 scale.
type LLMReranker struct {
	provider llm.Provider
}

// NewLLMReranker constructs an LLM-based reranker.
func NewLLMReranker(provider llm.Provider) *LLMReranker {
	return &LLMReranker{provider: provider}
}

const rerankerSystemPrompt = `You are a relevance scoring assistant. Given a query and a passage, rate the relevance of the passage to the query on a scale from 0 to 10.

- 0 = completely irrelevant
- 5 = somewhat relevant
- 10 = perfectly answers the query

Output ONLY the numeric score (integer 0-10), nothing else.`

// Rerank scores each block against the query using the LLM and sorts by score descending.
func (r *LLMReranker) Rerank(ctx context.Context, query string, blocks []EvidenceBlock) ([]EvidenceBlock, error) {
	if len(blocks) == 0 || r.provider == nil {
		return blocks, nil
	}

	type scoredBlock struct {
		block    EvidenceBlock
		llmScore float64
	}

	scored := make([]scoredBlock, len(blocks))
	for i, block := range blocks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := r.provider.Complete(ctx, llm.CompletionRequest{
			SystemPrompt: rerankerSystemPrompt,
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: fmt.Sprintf("Query: %s\n\nPassage: %s", query, block.Snippet),
				},
			},
			MaxTokens:   8,
			Temperature: 0.0,
		})
		if err != nil {
			slog.Warn("reranker llm call failed, keeping original score",
				slog.String("evidence_id", block.EvidenceID),
				slog.Any("error", err),
			)
			scored[i] = scoredBlock{block: block, llmScore: block.Score * 10}
			continue
		}

		score := parseRerankerScore(resp.Content)
		scored[i] = scoredBlock{block: block, llmScore: score}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].llmScore != scored[j].llmScore {
			return scored[i].llmScore > scored[j].llmScore
		}
		return scored[i].block.EvidenceID < scored[j].block.EvidenceID
	})

	result := make([]EvidenceBlock, len(scored))
	for i, s := range scored {
		b := s.block
		b.RerankScore = s.llmScore / 10.0 // normalize to [0,1]
		result[i] = b
	}

	return result, nil
}

func parseRerankerScore(content string) float64 {
	trimmed := strings.TrimSpace(content)
	score, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 5.0 // default to neutral score on parse failure
	}
	if score < 0 {
		return 0
	}
	if score > 10 {
		return 10
	}
	return score
}

// NoopReranker passes through blocks unchanged. Used when no reranker is configured.
type NoopReranker struct{}

// Rerank returns blocks unchanged.
func (r *NoopReranker) Rerank(_ context.Context, _ string, blocks []EvidenceBlock) ([]EvidenceBlock, error) {
	return blocks, nil
}

package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"opspilot-go/internal/llm"
)

const (
	rerankConcurrency    = 5
	rerankPerCallTimeout = 10 * time.Second
	rerankMaxBlocks      = 15
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
// Uses bounded concurrency and per-call timeouts to limit latency.
func (r *LLMReranker) Rerank(ctx context.Context, query string, blocks []EvidenceBlock) ([]EvidenceBlock, error) {
	if len(blocks) == 0 || r.provider == nil {
		return blocks, nil
	}

	// Cap the number of blocks to re-rank
	candidates := blocks
	if len(candidates) > rerankMaxBlocks {
		candidates = candidates[:rerankMaxBlocks]
	}

	type scoredBlock struct {
		block    EvidenceBlock
		llmScore float64
	}

	scored := make([]scoredBlock, len(candidates))
	sem := make(chan struct{}, rerankConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, block := range candidates {
		wg.Add(1)
		go func(idx int, b EvidenceBlock) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				mu.Lock()
				scored[idx] = scoredBlock{block: b, llmScore: 5.0}
				mu.Unlock()
				return
			}

			callCtx, cancel := context.WithTimeout(ctx, rerankPerCallTimeout)
			defer cancel()

			resp, err := r.provider.Complete(callCtx, llm.CompletionRequest{
				SystemPrompt: rerankerSystemPrompt,
				Messages: []llm.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Query: %s\n\nPassage: %s", query, b.Snippet),
					},
				},
				MaxTokens:   8,
				Temperature: 0.0,
			})
			if err != nil {
				slog.Warn("reranker llm call failed, using neutral score",
					slog.String("evidence_id", b.EvidenceID),
					slog.Any("error", err),
				)
				mu.Lock()
				scored[idx] = scoredBlock{block: b, llmScore: 5.0}
				mu.Unlock()
				return
			}

			score := parseRerankerScore(resp.Content)
			mu.Lock()
			scored[idx] = scoredBlock{block: b, llmScore: score}
			mu.Unlock()
		}(i, block)
	}
	wg.Wait()

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

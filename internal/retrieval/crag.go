package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"opspilot-go/internal/llm"
)

const (
	cragConcurrency    = 5
	cragPerCallTimeout = 10 * time.Second
)

// RelevanceVerdict is the CRAG classification for a retrieved passage.
type RelevanceVerdict string

const (
	// VerdictRelevant means the passage directly addresses the query.
	VerdictRelevant RelevanceVerdict = "relevant"
	// VerdictAmbiguous means the passage is partially related but may not fully answer.
	VerdictAmbiguous RelevanceVerdict = "ambiguous"
	// VerdictIrrelevant means the passage does not help answer the query.
	VerdictIrrelevant RelevanceVerdict = "irrelevant"
)

// CRAGFilter implements Corrective RAG (Yan et al., 2024): after retrieval,
// each passage is evaluated for relevance. Irrelevant passages are discarded,
// ambiguous ones are kept with a penalty, and only relevant passages proceed
// at full confidence.
type CRAGFilter struct {
	provider llm.Provider
}

// NewCRAGFilter constructs the corrective RAG filter.
func NewCRAGFilter(provider llm.Provider) *CRAGFilter {
	return &CRAGFilter{provider: provider}
}

const cragSystemPrompt = `You are a relevance classifier. Given a query and a passage, classify the passage into exactly one category:

- "relevant" — the passage directly helps answer the query
- "ambiguous" — the passage is partially related but may not fully answer
- "irrelevant" — the passage does not help answer the query

Output ONLY one word: relevant, ambiguous, or irrelevant.`

// Filter evaluates each evidence block for relevance and returns only
// relevant and ambiguous blocks. Irrelevant blocks are discarded.
// Ambiguous blocks have their Score penalized by 50%.
func (c *CRAGFilter) Filter(ctx context.Context, query string, blocks []EvidenceBlock) ([]EvidenceBlock, CRAGStats) {
	if len(blocks) == 0 || c.provider == nil {
		return blocks, CRAGStats{}
	}

	// Check if placeholder — skip filtering
	if _, ok := c.provider.(*llm.PlaceholderProvider); ok {
		return blocks, CRAGStats{Total: len(blocks), Relevant: len(blocks)}
	}

	type evaluated struct {
		block    EvidenceBlock
		verdict  RelevanceVerdict
		llmError bool
	}

	results := make([]evaluated, len(blocks))
	sem := make(chan struct{}, cragConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, block := range blocks {
		wg.Add(1)
		go func(idx int, b EvidenceBlock) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				mu.Lock()
				results[idx] = evaluated{block: b, verdict: VerdictAmbiguous}
				mu.Unlock()
				return
			}

			callCtx, cancel := context.WithTimeout(ctx, cragPerCallTimeout)
			defer cancel()

			verdict, classifyErr := c.classifyWithError(callCtx, query, b.Snippet)
			mu.Lock()
			results[idx] = evaluated{block: b, verdict: verdict, llmError: classifyErr != nil}
			mu.Unlock()
		}(i, block)
	}
	wg.Wait()

	var stats CRAGStats
	stats.Total = len(blocks)

	var filtered []EvidenceBlock
	for _, r := range results {
		if r.llmError {
			stats.Errors++
		}
		switch r.verdict {
		case VerdictRelevant:
			stats.Relevant++
			filtered = append(filtered, r.block)
		case VerdictAmbiguous:
			stats.Ambiguous++
			penalized := r.block
			penalized.Score *= 0.5
			filtered = append(filtered, penalized)
		case VerdictIrrelevant:
			stats.Irrelevant++
			slog.Debug("crag: discarded irrelevant passage",
				slog.String("evidence_id", r.block.EvidenceID),
			)
		}
	}

	return filtered, stats
}

func (c *CRAGFilter) classifyWithError(ctx context.Context, query, snippet string) (RelevanceVerdict, error) {
	resp, err := c.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: cragSystemPrompt,
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Query: %s\n\nPassage: %s", query, snippet),
			},
		},
		MaxTokens:   8,
		Temperature: 0.0,
	})
	if err != nil {
		slog.Warn("crag classification failed, defaulting to ambiguous",
			slog.Any("error", err),
		)
		return VerdictAmbiguous, err
	}

	return parseVerdict(resp.Content), nil
}

func parseVerdict(content string) RelevanceVerdict {
	lower := strings.ToLower(strings.TrimSpace(content))
	// Exact match first (most reliable)
	switch lower {
	case "relevant":
		return VerdictRelevant
	case "irrelevant":
		return VerdictIrrelevant
	case "ambiguous":
		return VerdictAmbiguous
	}
	// Fallback: contains-based matching with negation awareness
	switch {
	case strings.Contains(lower, "irrelevant") || strings.Contains(lower, "not relevant"):
		return VerdictIrrelevant
	case strings.Contains(lower, "relevant"):
		return VerdictRelevant
	case strings.Contains(lower, "ambiguous"):
		return VerdictAmbiguous
	default:
		return VerdictAmbiguous
	}
}

// CRAGStats tracks the distribution of CRAG verdicts for observability.
type CRAGStats struct {
	Total      int
	Relevant   int
	Ambiguous  int
	Irrelevant int
	Errors     int // LLM call failures (counted within Ambiguous)
}

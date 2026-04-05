// Package retrieval — ContextualCompressor implements the LangChain
// ContextualCompressionRetriever pattern: each evidence snippet is compressed
// by an LLM to extract only the parts relevant to the query. This reduces
// token waste in the LLM context window and improves answer grounding by
// eliminating irrelevant passage content.
//
// Reference: LangChain ContextualCompressionRetriever (2023)
// Pattern: query + passage → LLM → compressed passage (or empty if irrelevant)

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

const compressorPrompt = `You are a passage compressor. Given a user query and a retrieved passage, extract ONLY the sentences from the passage that are directly relevant to answering the query. Remove irrelevant content.

Rules:
1. Return only the relevant sentences from the passage, preserving their original wording.
2. If NO part of the passage is relevant to the query, return exactly: IRRELEVANT
3. Do not add any explanation, commentary, or new information.
4. Do not rephrase — use the original text.
5. Be aggressive: remove anything that doesn't directly help answer the query.`

// ContextualCompressor compresses evidence blocks by extracting only
// query-relevant content from each passage via LLM.
type ContextualCompressor struct {
	llm         llm.Provider
	concurrency int
	timeout     time.Duration
}

// NewContextualCompressor constructs a compressor with bounded concurrency.
func NewContextualCompressor(provider llm.Provider) *ContextualCompressor {
	if provider == nil {
		return nil
	}
	if _, isPlaceholder := provider.(*llm.PlaceholderProvider); isPlaceholder {
		return nil
	}
	return &ContextualCompressor{
		llm:         provider,
		concurrency: 5,
		timeout:     8 * time.Second,
	}
}

// Compress processes each evidence block through the LLM to extract only
// query-relevant content. Blocks classified as IRRELEVANT are removed.
// Returns compressed blocks preserving original provenance metadata.
func (c *ContextualCompressor) Compress(ctx context.Context, query string, blocks []EvidenceBlock) []EvidenceBlock {
	if c == nil || len(blocks) == 0 {
		return blocks
	}

	type result struct {
		idx       int
		block     EvidenceBlock
		keep      bool
	}

	results := make([]result, len(blocks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.concurrency)

	for i, block := range blocks {
		wg.Add(1)
		go func(idx int, b EvidenceBlock) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			compressed, keep := c.compressOne(ctx, query, b)
			results[idx] = result{idx: idx, block: compressed, keep: keep}
		}(i, block)
	}

	wg.Wait()

	out := make([]EvidenceBlock, 0, len(blocks))
	for _, r := range results {
		if r.keep {
			out = append(out, r.block)
		}
	}

	slog.Debug("contextual compression completed",
		slog.Int("input_blocks", len(blocks)),
		slog.Int("output_blocks", len(out)),
		slog.Int("removed", len(blocks)-len(out)),
	)

	return out
}

func (c *ContextualCompressor) compressOne(ctx context.Context, query string, block EvidenceBlock) (EvidenceBlock, bool) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	userMsg := fmt.Sprintf("Query: %s\n\nPassage [%s] %s:\n%s",
		query, block.CitationLabel, block.SourceTitle, block.Snippet)

	resp, err := c.llm.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt: compressorPrompt,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    512,
		Temperature:  llm.TemperaturePtr(0),
	})
	if err != nil {
		slog.Warn("contextual compression failed for block, keeping original",
			slog.String("evidence_id", block.EvidenceID),
			slog.Any("error", err),
		)
		return block, true
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" || strings.EqualFold(content, "IRRELEVANT") {
		return block, false
	}

	// Preserve all provenance metadata, only replace the snippet text
	compressed := block
	compressed.Snippet = content
	return compressed, true
}

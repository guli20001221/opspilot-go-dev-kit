package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"opspilot-go/internal/llm"
)

const hydePerCallTimeout = 10 * time.Second

// HyDERewriter implements Hypothetical Document Embeddings (Gao et al., 2022):
// before retrieval, the LLM generates a hypothetical answer to the query.
// That hypothetical answer is used as the retrieval query instead of the raw
// user question, improving semantic matching because the hypothetical document
// lives in the same embedding space as the stored passages.
type HyDERewriter struct {
	provider llm.Provider
}

// NewHyDERewriter constructs a HyDE query rewriter.
func NewHyDERewriter(provider llm.Provider) *HyDERewriter {
	return &HyDERewriter{provider: provider}
}

const hydeSystemPrompt = `You are a knowledge base assistant. Given a user question, write a short passage (2-4 sentences) that would be the ideal answer found in an internal knowledge base document. Do not say "I don't know" or ask clarifying questions. Just write the passage as if it existed in the documentation. Output ONLY the passage, nothing else.`

// Rewrite generates a hypothetical document for the given query.
// If the provider is nil, a placeholder, or the call fails, the original
// query is returned unchanged so the retrieval pipeline degrades gracefully.
func (h *HyDERewriter) Rewrite(ctx context.Context, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", nil
	}

	if h.provider == nil {
		return query, nil
	}

	// Placeholder provider: skip the LLM call, return query as-is
	if _, ok := h.provider.(*llm.PlaceholderProvider); ok {
		return query, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, hydePerCallTimeout)
	defer cancel()

	resp, err := h.provider.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt: hydeSystemPrompt,
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Question: %s", query),
			},
		},
		MaxTokens:   256,
		Temperature: llm.TemperaturePtr(0.7),
	})
	if err != nil {
		slog.Warn("hyde rewrite failed, using original query",
			slog.Any("error", err),
		)
		return query, nil
	}

	rewritten := strings.TrimSpace(resp.Content)
	if rewritten == "" {
		return query, nil
	}

	slog.Debug("hyde rewrite applied",
		slog.String("original_query", query),
		slog.Int("hypothetical_len", len(rewritten)),
	)

	return rewritten, nil
}

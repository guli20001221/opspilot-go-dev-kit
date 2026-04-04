package ingestion

import (
	"context"
	"fmt"
	"sync"

	"opspilot-go/internal/llm"
)

const contextPrefixSystemPrompt = `You are a document analysis assistant. Given a document and one of its chunks, generate a single concise sentence that situates this chunk within the broader document context. Do not repeat the chunk content itself. Output only the context sentence, nothing else.`

const maxDocContextChars = 8000

// PrefixerOptions configures the contextual prefixer.
type PrefixerOptions struct {
	MaxTokens   int // LLM max tokens for prefix generation; default 128
	Concurrency int // parallel LLM calls; default 3
}

// ContextPrefixer generates contextual prefixes for chunks using an LLM.
type ContextPrefixer struct {
	llm         llm.Provider
	maxTokens   int
	concurrency int
}

// NewContextPrefixer constructs the contextual prefixer.
func NewContextPrefixer(provider llm.Provider, opts PrefixerOptions) *ContextPrefixer {
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 128
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 3
	}
	return &ContextPrefixer{
		llm:         provider,
		maxTokens:   opts.MaxTokens,
		concurrency: opts.Concurrency,
	}
}

// Prefix generates contextual prefixes for each chunk.
func (p *ContextPrefixer) Prefix(ctx context.Context, doc Document, chunks []Chunk) ([]Chunk, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	docContext := doc.Content
	if len(docContext) > maxDocContextChars {
		docContext = docContext[:maxDocContextChars] + "\n[...truncated]"
	}

	result := make([]Chunk, len(chunks))
	copy(result, chunks)

	if p.llm == nil {
		// Deterministic fallback without LLM
		for i := range result {
			result[i].ContextPrefix = fmt.Sprintf("This chunk is from document: %s.", doc.SourceTitle)
		}
		return result, nil
	}

	// Check if placeholder provider — use deterministic prefix
	if _, ok := p.llm.(*llm.PlaceholderProvider); ok {
		for i := range result {
			result[i].ContextPrefix = fmt.Sprintf("This chunk is from document: %s.", doc.SourceTitle)
		}
		return result, nil
	}

	// Bounded concurrency for real LLM calls
	sem := make(chan struct{}, p.concurrency)
	var mu sync.Mutex
	var firstErr error

	var wg sync.WaitGroup
	for i := range result {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			prefix, err := p.generatePrefix(ctx, docContext, result[idx].Text)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				result[idx].ContextPrefix = fmt.Sprintf("This chunk is from document: %s.", doc.SourceTitle)
				return
			}
			result[idx].ContextPrefix = prefix
		}(i)
	}
	wg.Wait()

	// Don't fail the pipeline on prefix errors — degraded prefixes are acceptable
	return result, nil
}

func (p *ContextPrefixer) generatePrefix(ctx context.Context, docContext, chunkText string) (string, error) {
	resp, err := p.llm.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: contextPrefixSystemPrompt,
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Document:\n%s\n\nChunk:\n%s", docContext, chunkText),
			},
		},
		MaxTokens: p.maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

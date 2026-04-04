package llm

import "context"

// Provider generates text completions from a prompt and message history.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// Message is one conversation turn sent to the LLM provider.
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// CompletionRequest is the typed input for a single LLM completion call.
type CompletionRequest struct {
	SystemPrompt string
	Messages     []Message
	Model        string  // override; empty = use adapter default
	MaxTokens    int     // 0 = provider default
	Temperature  float64 // 0 = provider default
}

// CompletionResponse is the typed output from one LLM completion call.
type CompletionResponse struct {
	Content      string
	Model        string
	PromptTokens int
	OutputTokens int
}

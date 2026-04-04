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

// ResponseFormat constants control the output format requested from the provider.
const (
	// ResponseFormatText requests the default free-form text output.
	ResponseFormatText = ""
	// ResponseFormatJSON requests a JSON object output (structured output / JSON mode).
	ResponseFormatJSON = "json_object"
)

// CompletionRequest is the typed input for a single LLM completion call.
type CompletionRequest struct {
	SystemPrompt   string
	Messages       []Message
	Model          string  // override; empty = use adapter default
	MaxTokens      int     // 0 = provider default
	Temperature    float64 // 0 = provider default
	ResponseFormat string  // "" = text, "json_object" = JSON mode
}

// CompletionResponse is the typed output from one LLM completion call.
type CompletionResponse struct {
	Content      string
	Model        string
	PromptTokens int
	OutputTokens int
}

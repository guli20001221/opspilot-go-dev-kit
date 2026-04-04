package llm

import "context"

// PlaceholderContent is the default response when no real provider is configured.
const PlaceholderContent = "Milestone 1 placeholder response."

// PlaceholderProvider returns a static placeholder response.
type PlaceholderProvider struct{}

// NewPlaceholderProvider constructs the no-op placeholder provider.
func NewPlaceholderProvider() *PlaceholderProvider {
	return &PlaceholderProvider{}
}

// Complete returns the placeholder response without making any external calls.
func (p *PlaceholderProvider) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{
		Content: PlaceholderContent,
		Model:   "placeholder",
	}, nil
}

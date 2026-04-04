package retrieval

import (
	"fmt"
	"time"
)

// EmbedderOptions configures the embedding provider factory.
type EmbedderOptions struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// NewConfiguredEmbedder constructs the appropriate embedder from configuration.
func NewConfiguredEmbedder(opts EmbedderOptions) (Embedder, error) {
	switch opts.Provider {
	case "", "placeholder":
		return &PlaceholderEmbedder{}, nil
	case "openai":
		if opts.BaseURL == "" {
			return nil, fmt.Errorf("openai embedder requires OPSPILOT_EMBEDDING_BASE_URL")
		}
		if opts.Model == "" {
			return nil, fmt.Errorf("openai embedder requires OPSPILOT_EMBEDDING_MODEL")
		}
		return NewOpenAIEmbedder(OpenAIEmbedderOptions{
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
			Model:   opts.Model,
			Timeout: opts.Timeout,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", opts.Provider)
	}
}

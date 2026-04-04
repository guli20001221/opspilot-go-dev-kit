package llm

import (
	"fmt"
	"time"
)

// ProviderOptions configures the LLM provider factory.
type ProviderOptions struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// NewConfiguredProvider constructs the appropriate LLM provider from configuration.
func NewConfiguredProvider(opts ProviderOptions) (Provider, error) {
	switch opts.Provider {
	case "", "placeholder":
		return NewPlaceholderProvider(), nil
	case "openai":
		if opts.BaseURL == "" {
			return nil, fmt.Errorf("openai provider requires OPSPILOT_LLM_BASE_URL")
		}
		if opts.Model == "" {
			return nil, fmt.Errorf("openai provider requires OPSPILOT_LLM_MODEL")
		}
		return NewOpenAIProvider(OpenAIOptions{
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
			Model:   opts.Model,
			Timeout: opts.Timeout,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q", opts.Provider)
	}
}

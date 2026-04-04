package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIOptions configures the OpenAI-compatible HTTP adapter.
type OpenAIOptions struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
	Client  *http.Client
}

// OpenAIProvider calls an OpenAI-compatible chat completions API.
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAIProvider constructs the OpenAI-compatible adapter.
func NewOpenAIProvider(opts OpenAIOptions) *OpenAIProvider {
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &OpenAIProvider{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		apiKey:  opts.APIKey,
		model:   opts.Model,
		client:  client,
	}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
	Model   string         `json:"model"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Complete sends a chat completion request and returns the first choice.
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]openAIMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, m := range req.Messages {
		messages = append(messages, openAIMessage{Role: m.Role, Content: m.Content})
	}

	body := openAIRequest{
		Model:    model,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		body.Temperature = &req.Temperature
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("provider request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("provider status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("provider returned empty choices")
	}

	return CompletionResponse{
		Content:      result.Choices[0].Message.Content,
		Model:        result.Model,
		PromptTokens: result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
	}, nil
}

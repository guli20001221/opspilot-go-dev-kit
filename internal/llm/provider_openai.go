package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opspilot-go/internal/observability/tracing"
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

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIMessage       `json:"messages"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Temperature    *float64              `json:"temperature,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
	Stream         bool                  `json:"stream,omitempty"`
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
	ctx, span := tracing.StartSpan(ctx, "llm.complete",
		tracing.AttrProvider.String("openai"),
	)
	defer span.End()

	model := req.Model
	if model == "" {
		model = p.model
	}
	span.SetAttributes(tracing.AttrModel.String(model))

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
	if req.Temperature != nil {
		body.Temperature = req.Temperature
	}
	if req.ResponseFormat == ResponseFormatJSON {
		body.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
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

	const maxResponseBytes = 10 * 1024 * 1024 // 10 MB
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBytes))
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

	span.SetAttributes(
		tracing.AttrTokensIn.Int(result.Usage.PromptTokens),
		tracing.AttrTokensOut.Int(result.Usage.CompletionTokens),
	)

	return CompletionResponse{
		Content:      result.Choices[0].Message.Content,
		Model:        result.Model,
		PromptTokens: result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
	}, nil
}

// StreamComplete sends a streaming chat completion request and calls onToken for each delta.
func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest, onToken func(token string)) (CompletionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "llm.stream_complete",
		tracing.AttrProvider.String("openai"),
	)
	defer span.End()

	model := req.Model
	if model == "" {
		model = p.model
	}
	span.SetAttributes(tracing.AttrModel.String(model))

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
		Stream:   true,
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		body.Temperature = req.Temperature
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal stream request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("stream request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
		return CompletionResponse{}, fmt.Errorf("stream status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse SSE stream
	var fullContent strings.Builder
	var respModel string
	scanner := bufio.NewScanner(httpResp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if respModel == "" && chunk.Model != "" {
			respModel = chunk.Model
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if onToken != nil {
					onToken(choice.Delta.Content)
				}
			}
		}
	}

	span.SetAttributes(tracing.AttrModel.String(respModel))

	return CompletionResponse{
		Content: fullContent.String(),
		Model:   respModel,
	}, nil
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Model   string         `json:"model"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamDelta struct {
	Content string `json:"content"`
}

// Verify OpenAIProvider implements StreamingProvider.
var _ StreamingProvider = (*OpenAIProvider)(nil)

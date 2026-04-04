package retrieval

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

// OpenAIEmbedderOptions configures the OpenAI-compatible embedding adapter.
type OpenAIEmbedderOptions struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
	Client  *http.Client
}

// OpenAIEmbedder calls an OpenAI-compatible embeddings API.
type OpenAIEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAIEmbedder constructs the OpenAI-compatible embedding adapter.
func NewOpenAIEmbedder(opts OpenAIEmbedderOptions) *OpenAIEmbedder {
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &OpenAIEmbedder{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		apiKey:  opts.APIKey,
		model:   opts.Model,
		client:  client,
	}
}

type embeddingRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	EncodingFormat string `json:"encoding_format"`
}

type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Usage embeddingUsage  `json:"usage"`
	Model string          `json:"model"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type embeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Embed generates a vector embedding for the input text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	payload, err := json.Marshal(embeddingRequest{
		Model:          e.model,
		Input:          text,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	url := e.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	httpResp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer httpResp.Body.Close()

	const maxResponseBytes = 10 * 1024 * 1024
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal embedding response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding response returned empty data")
	}

	return result.Data[0].Embedding, nil
}

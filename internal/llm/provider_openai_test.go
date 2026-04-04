package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("model = %q, want test-model", req.Model)
		}
		if len(req.Messages) < 2 {
			t.Fatalf("len(messages) = %d, want >= 2", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Fatalf("messages[0].role = %q, want system", req.Messages[0].Role)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{{Message: openAIMessage{Role: "assistant", Content: "Hello from LLM"}}},
			Usage:   openAIUsage{PromptTokens: 10, CompletionTokens: 5},
			Model:   "test-model",
		})
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIOptions{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	resp, err := p.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "You are a test assistant.",
		Messages:     []Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Hello from LLM" {
		t.Fatalf("Content = %q, want %q", resp.Content, "Hello from LLM")
	}
	if resp.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", resp.PromptTokens)
	}
	if resp.OutputTokens != 5 {
		t.Fatalf("OutputTokens = %d, want 5", resp.OutputTokens)
	}
}

func TestOpenAIProviderSendsTemperatureZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Temperature == nil {
			t.Fatal("temperature is nil, want explicit zero")
		}
		if *req.Temperature != 0 {
			t.Fatalf("temperature = %f, want 0", *req.Temperature)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{{Message: openAIMessage{Role: "assistant", Content: "ok"}}},
			Model:   "test-model",
		})
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIOptions{BaseURL: server.URL, APIKey: "k", Model: "m"})
	temp := 0.0
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		Temperature: &temp,
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestOpenAIProviderOmitsTemperatureWhenNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := raw["temperature"]; ok {
			t.Fatal("temperature should be omitted when nil, but it was present")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{{Message: openAIMessage{Role: "assistant", Content: "ok"}}},
			Model:   "test-model",
		})
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIOptions{BaseURL: server.URL, APIKey: "k", Model: "m"})
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Hi"}},
		// Temperature not set — should be nil, omitted from JSON
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestOpenAIProviderNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIOptions{BaseURL: server.URL, Model: "m"})
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("Complete() error = nil, want non-nil")
	}
}

func TestOpenAIProviderEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{Choices: []openAIChoice{}})
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIOptions{BaseURL: server.URL, Model: "m"})
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("Complete() error = nil, want non-nil for empty choices")
	}
}

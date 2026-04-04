package retrieval

import (
	"context"
	"testing"

	"opspilot-go/internal/llm"
)

type mockRerankerProvider struct {
	scores map[string]string // snippet → score response
}

func (m *mockRerankerProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	for _, msg := range req.Messages {
		for snippet, score := range m.scores {
			if contains(msg.Content, snippet) {
				return llm.CompletionResponse{Content: score}, nil
			}
		}
	}
	return llm.CompletionResponse{Content: "5"}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLLMRerankerSortsByScore(t *testing.T) {
	provider := &mockRerankerProvider{
		scores: map[string]string{
			"password reset": "9",
			"refund policy":  "3",
			"account setup":  "7",
		},
	}

	reranker := NewLLMReranker(provider)
	blocks := []EvidenceBlock{
		{EvidenceID: "a", Snippet: "refund policy details"},
		{EvidenceID: "b", Snippet: "password reset guide"},
		{EvidenceID: "c", Snippet: "account setup steps"},
	}

	result, err := reranker.Rerank(context.Background(), "how to reset password", blocks)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	// Should be sorted: password (9) > account (7) > refund (3)
	if result[0].EvidenceID != "b" {
		t.Fatalf("result[0] = %q, want b (password, score 9)", result[0].EvidenceID)
	}
	if result[1].EvidenceID != "c" {
		t.Fatalf("result[1] = %q, want c (account, score 7)", result[1].EvidenceID)
	}
	if result[2].EvidenceID != "a" {
		t.Fatalf("result[2] = %q, want a (refund, score 3)", result[2].EvidenceID)
	}

	// Verify rerank scores are normalized to [0,1]
	if result[0].RerankScore < 0.8 || result[0].RerankScore > 1.0 {
		t.Fatalf("result[0].RerankScore = %f, want ~0.9", result[0].RerankScore)
	}
}

func TestNoopRerankerPassesThrough(t *testing.T) {
	reranker := &NoopReranker{}
	blocks := []EvidenceBlock{{EvidenceID: "a"}, {EvidenceID: "b"}}
	result, err := reranker.Rerank(context.Background(), "query", blocks)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(result) != 2 || result[0].EvidenceID != "a" {
		t.Fatalf("result = %v, want passthrough", result)
	}
}

func TestLLMRerankerHandlesNilProvider(t *testing.T) {
	reranker := NewLLMReranker(nil)
	blocks := []EvidenceBlock{{EvidenceID: "a"}}
	result, err := reranker.Rerank(context.Background(), "query", blocks)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
}

func TestParseRerankerScore(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"8", 8},
		{"  7  ", 7},
		{"0", 0},
		{"10", 10},
		{"-1", 0},
		{"15", 10},
		{"not a number", 5},
		{"", 5},
	}
	for _, tt := range tests {
		got := parseRerankerScore(tt.input)
		if got != tt.want {
			t.Errorf("parseRerankerScore(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

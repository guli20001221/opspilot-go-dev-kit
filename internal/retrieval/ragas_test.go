package retrieval

import (
	"context"
	"strings"
	"testing"

	"opspilot-go/internal/llm"
)

type mockRAGASProvider struct {
	responses map[string]string // prompt keyword → score response
}

func (m *mockRAGASProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	for keyword, score := range m.responses {
		if strings.Contains(req.SystemPrompt, keyword) {
			return llm.CompletionResponse{Content: score}, nil
		}
	}
	return llm.CompletionResponse{Content: "0.5"}, nil
}

func TestRAGASEvaluatorComputesAllMetrics(t *testing.T) {
	provider := &mockRAGASProvider{
		responses: map[string]string{
			"faithfulness": "0.9",
			"relevancy":    "0.8",
			"precision":    "0.7",
		},
	}

	evaluator := NewRAGASEvaluator(provider)
	metrics, err := evaluator.Evaluate(context.Background(), RAGASInput{
		Query:    "How do I reset my password?",
		Answer:   "Navigate to Settings > Security > Reset Password.",
		Contexts: []string{"To reset your password, go to Settings and find Security."},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if metrics.Faithfulness != 0.9 {
		t.Fatalf("Faithfulness = %f, want 0.9", metrics.Faithfulness)
	}
	if metrics.AnswerRelevancy != 0.8 {
		t.Fatalf("AnswerRelevancy = %f, want 0.8", metrics.AnswerRelevancy)
	}
	if metrics.ContextPrecision != 0.7 {
		t.Fatalf("ContextPrecision = %f, want 0.7", metrics.ContextPrecision)
	}
	if metrics.OverallScore <= 0 || metrics.OverallScore > 1 {
		t.Fatalf("OverallScore = %f, want in (0,1]", metrics.OverallScore)
	}
}

func TestRAGASEvaluatorNilProviderReturnsError(t *testing.T) {
	evaluator := NewRAGASEvaluator(nil)
	_, err := evaluator.Evaluate(context.Background(), RAGASInput{Query: "test"})
	if err == nil {
		t.Fatal("Evaluate() error = nil, want non-nil for nil provider")
	}
}

func TestHarmonicMean(t *testing.T) {
	tests := []struct {
		values []float64
		want   float64
	}{
		{[]float64{1, 1, 1}, 1.0},
		{[]float64{0, 0.5, 1}, 0.0},
		{[]float64{}, 0.0},
	}
	for _, tt := range tests {
		got := harmonicMean(tt.values...)
		if got != tt.want {
			t.Errorf("harmonicMean(%v) = %f, want %f", tt.values, got, tt.want)
		}
	}
}

func TestParseScore01(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0.85", 0.85},
		{"1.0", 1.0},
		{"0.0", 0.0},
		{"-0.5", 0},
		{"1.5", 1},
		{"not a number", 0.5},
		{"  0.7  ", 0.7},
	}
	for _, tt := range tests {
		got := parseScore01(tt.input)
		if got != tt.want {
			t.Errorf("parseScore01(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

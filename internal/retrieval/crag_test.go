package retrieval

import (
	"context"
	"strings"
	"testing"

	"opspilot-go/internal/llm"
)

type mockCRAGProvider struct {
	verdicts map[string]string // snippet keyword → verdict
}

func (m *mockCRAGProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	for _, msg := range req.Messages {
		for keyword, verdict := range m.verdicts {
			if strings.Contains(msg.Content, keyword) {
				return llm.CompletionResponse{Content: verdict}, nil
			}
		}
	}
	return llm.CompletionResponse{Content: "ambiguous"}, nil
}

func TestCRAGFilterDiscardsIrrelevant(t *testing.T) {
	provider := &mockCRAGProvider{
		verdicts: map[string]string{
			"reset guide":      "relevant",
			"weather forecast": "irrelevant",
			"maybe reset":      "ambiguous",
		},
	}

	filter := NewCRAGFilter(provider)
	blocks := []EvidenceBlock{
		{EvidenceID: "a", Snippet: "password reset guide", Score: 0.9},
		{EvidenceID: "b", Snippet: "weather forecast today", Score: 0.7},
		{EvidenceID: "c", Snippet: "maybe reset your account", Score: 0.6},
	}

	result, stats := filter.Filter(context.Background(), "how do I change my credentials", blocks)

	if stats.Total != 3 {
		t.Fatalf("stats.Total = %d, want 3", stats.Total)
	}
	if stats.Relevant != 1 {
		t.Fatalf("stats.Relevant = %d, want 1", stats.Relevant)
	}
	if stats.Irrelevant != 1 {
		t.Fatalf("stats.Irrelevant = %d, want 1", stats.Irrelevant)
	}
	if stats.Ambiguous != 1 {
		t.Fatalf("stats.Ambiguous = %d, want 1", stats.Ambiguous)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2 (irrelevant discarded)", len(result))
	}
	// Verify relevant block kept original score
	if result[0].EvidenceID != "a" || result[0].Score != 0.9 {
		t.Fatalf("result[0] = %q score=%f, want a/0.9", result[0].EvidenceID, result[0].Score)
	}
	// Verify ambiguous block score penalized by 50%
	if result[1].EvidenceID != "c" || result[1].Score != 0.3 {
		t.Fatalf("result[1] = %q score=%f, want c/0.3", result[1].EvidenceID, result[1].Score)
	}
}

func TestCRAGFilterNilProvider(t *testing.T) {
	filter := NewCRAGFilter(nil)
	blocks := []EvidenceBlock{{EvidenceID: "a"}}
	result, stats := filter.Filter(context.Background(), "query", blocks)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1 (passthrough)", len(result))
	}
	if stats.Total != 0 {
		t.Fatalf("stats.Total = %d, want 0 (skipped)", stats.Total)
	}
}

func TestCRAGFilterPlaceholderProvider(t *testing.T) {
	filter := NewCRAGFilter(llm.NewPlaceholderProvider())
	blocks := []EvidenceBlock{{EvidenceID: "a"}, {EvidenceID: "b"}}
	result, stats := filter.Filter(context.Background(), "query", blocks)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2 (placeholder passthrough)", len(result))
	}
	if stats.Relevant != 2 {
		t.Fatalf("stats.Relevant = %d, want 2", stats.Relevant)
	}
}

func TestParseVerdict(t *testing.T) {
	tests := []struct {
		input string
		want  RelevanceVerdict
	}{
		{"relevant", VerdictRelevant},
		{"Relevant", VerdictRelevant},
		{"irrelevant", VerdictIrrelevant},
		{"IRRELEVANT", VerdictIrrelevant},
		{"ambiguous", VerdictAmbiguous},
		{"not sure", VerdictAmbiguous},
		{"", VerdictAmbiguous},
		{"The passage is relevant to the query.", VerdictRelevant},
		{"This is irrelevant.", VerdictIrrelevant},
	}
	for _, tt := range tests {
		got := parseVerdict(tt.input)
		if got != tt.want {
			t.Errorf("parseVerdict(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

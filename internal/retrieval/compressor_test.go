package retrieval

import (
	"context"
	"strings"
	"testing"

	"opspilot-go/internal/llm"
)

type mockCompressorLLM struct {
	responses map[string]string // snippet substring → compressed response
}

func (m *mockCompressorLLM) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	userMsg := req.Messages[0].Content
	for key, resp := range m.responses {
		if strings.Contains(userMsg, key) {
			return llm.CompletionResponse{Content: resp, Model: "mock"}, nil
		}
	}
	return llm.CompletionResponse{Content: "IRRELEVANT", Model: "mock"}, nil
}

func TestContextualCompressorCompressesRelevantBlocks(t *testing.T) {
	provider := &mockCompressorLLM{
		responses: map[string]string{
			"Password Guide": "Navigate to Settings and find Security. Click Reset Password.",
			"Recovery Guide":  "If you cannot access your email, contact support.",
		},
	}
	compressor := NewContextualCompressor(provider)

	blocks := []EvidenceBlock{
		{EvidenceID: "ev-1", SourceTitle: "Password Guide", Snippet: "How to reset your password. Navigate to Settings and find Security. Click Reset Password and follow the prompts. Additional unrelated info here.", CitationLabel: "[1]", Score: 0.9},
		{EvidenceID: "ev-2", SourceTitle: "Recovery Guide", Snippet: "Account recovery options. If you cannot access your email, contact support. Provide your account ID.", CitationLabel: "[2]", Score: 0.8},
	}

	got := compressor.Compress(context.Background(), "How do I reset my password?", blocks)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	// Snippets should be compressed (shorter than original)
	if got[0].Snippet == blocks[0].Snippet {
		t.Fatal("first block snippet should be compressed, got original")
	}
	// Provenance preserved
	if got[0].EvidenceID != "ev-1" || got[0].SourceTitle != "Password Guide" {
		t.Fatalf("provenance lost: %+v", got[0])
	}
	if got[1].EvidenceID != "ev-2" {
		t.Fatalf("provenance lost: %+v", got[1])
	}
}

func TestContextualCompressorRemovesIrrelevantBlocks(t *testing.T) {
	provider := &mockCompressorLLM{
		responses: map[string]string{
			"reset your password": "Reset via Settings > Security.",
			// "weather" not in responses → default IRRELEVANT
		},
	}
	compressor := NewContextualCompressor(provider)

	blocks := []EvidenceBlock{
		{EvidenceID: "ev-1", Snippet: "How to reset your password.", CitationLabel: "[1]", Score: 0.9},
		{EvidenceID: "ev-2", Snippet: "Today's weather forecast is sunny.", CitationLabel: "[2]", Score: 0.3},
	}

	got := compressor.Compress(context.Background(), "password reset", blocks)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 (irrelevant block removed)", len(got))
	}
	if got[0].EvidenceID != "ev-1" {
		t.Fatalf("wrong block kept: %+v", got[0])
	}
}

func TestContextualCompressorNilSafe(t *testing.T) {
	var compressor *ContextualCompressor
	blocks := []EvidenceBlock{{EvidenceID: "ev-1", Snippet: "test"}}
	got := compressor.Compress(context.Background(), "query", blocks)
	if len(got) != 1 {
		t.Fatalf("nil compressor should return blocks unchanged, got %d", len(got))
	}
}

func TestContextualCompressorPlaceholderProviderReturnsNil(t *testing.T) {
	compressor := NewContextualCompressor(llm.NewPlaceholderProvider())
	if compressor != nil {
		t.Fatal("placeholder provider should return nil compressor")
	}
}

func TestContextualCompressorEmptyBlocks(t *testing.T) {
	provider := &mockCompressorLLM{}
	compressor := NewContextualCompressor(provider)
	got := compressor.Compress(context.Background(), "query", nil)
	if got != nil {
		t.Fatalf("empty input should return nil, got %v", got)
	}
}

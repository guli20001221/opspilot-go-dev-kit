package retrieval

import (
	"context"
	"errors"
	"testing"

	"opspilot-go/internal/llm"
)

type mockHyDEProvider struct {
	response string
	err      error
}

func (m *mockHyDEProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	if m.err != nil {
		return llm.CompletionResponse{}, m.err
	}
	return llm.CompletionResponse{Content: m.response}, nil
}

func TestHyDERewriterGeneratesHypotheticalDocument(t *testing.T) {
	provider := &mockHyDEProvider{
		response: "To reset your password, navigate to Settings > Security > Change Password.",
	}
	rewriter := NewHyDERewriter(provider)

	got := rewriter.Rewrite(context.Background(), "how do I reset my password")
	if got != provider.response {
		t.Fatalf("Rewrite() = %q, want hypothetical document", got)
	}
}

func TestHyDERewriterNilProviderPassthrough(t *testing.T) {
	rewriter := NewHyDERewriter(nil)
	got := rewriter.Rewrite(context.Background(), "how do I reset my password")
	if got != "how do I reset my password" {
		t.Fatalf("Rewrite() = %q, want original query", got)
	}
}

func TestHyDERewriterPlaceholderProviderPassthrough(t *testing.T) {
	rewriter := NewHyDERewriter(llm.NewPlaceholderProvider())
	got := rewriter.Rewrite(context.Background(), "ticket status")
	if got != "ticket status" {
		t.Fatalf("Rewrite() = %q, want original query", got)
	}
}

func TestHyDERewriterEmptyQueryReturnsEmpty(t *testing.T) {
	rewriter := NewHyDERewriter(&mockHyDEProvider{response: "should not be used"})
	got := rewriter.Rewrite(context.Background(), "  ")
	if got != "" {
		t.Fatalf("Rewrite() = %q, want empty string", got)
	}
}

func TestHyDERewriterLLMErrorFallsBackToOriginal(t *testing.T) {
	provider := &mockHyDEProvider{err: errors.New("provider unavailable")}
	rewriter := NewHyDERewriter(provider)
	got := rewriter.Rewrite(context.Background(), "incident runbook")
	if got != "incident runbook" {
		t.Fatalf("Rewrite() = %q, want original query on LLM error", got)
	}
}

func TestHyDERewriterEmptyResponseFallsBackToOriginal(t *testing.T) {
	provider := &mockHyDEProvider{response: "  "}
	rewriter := NewHyDERewriter(provider)
	got := rewriter.Rewrite(context.Background(), "deployment steps")
	if got != "deployment steps" {
		t.Fatalf("Rewrite() = %q, want original query when LLM returns empty", got)
	}
}

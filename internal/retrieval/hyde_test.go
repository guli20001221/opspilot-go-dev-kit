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
		response: "To reset your password, navigate to Settings > Security > Change Password. Enter your current password and then your new password twice to confirm.",
	}
	rewriter := NewHyDERewriter(provider)

	got, err := rewriter.Rewrite(context.Background(), "how do I reset my password")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != provider.response {
		t.Fatalf("Rewrite() = %q, want hypothetical document", got)
	}
}

func TestHyDERewriterNilProviderPassthrough(t *testing.T) {
	rewriter := NewHyDERewriter(nil)

	got, err := rewriter.Rewrite(context.Background(), "how do I reset my password")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != "how do I reset my password" {
		t.Fatalf("Rewrite() = %q, want original query", got)
	}
}

func TestHyDERewriterPlaceholderProviderPassthrough(t *testing.T) {
	rewriter := NewHyDERewriter(llm.NewPlaceholderProvider())

	got, err := rewriter.Rewrite(context.Background(), "ticket status")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != "ticket status" {
		t.Fatalf("Rewrite() = %q, want original query", got)
	}
}

func TestHyDERewriterEmptyQueryReturnsEmpty(t *testing.T) {
	rewriter := NewHyDERewriter(&mockHyDEProvider{response: "should not be used"})

	got, err := rewriter.Rewrite(context.Background(), "  ")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != "" {
		t.Fatalf("Rewrite() = %q, want empty string", got)
	}
}

func TestHyDERewriterLLMErrorFallsBackToOriginal(t *testing.T) {
	provider := &mockHyDEProvider{err: errors.New("provider unavailable")}
	rewriter := NewHyDERewriter(provider)

	got, err := rewriter.Rewrite(context.Background(), "incident runbook")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != "incident runbook" {
		t.Fatalf("Rewrite() = %q, want original query on LLM error", got)
	}
}

func TestHyDERewriterEmptyResponseFallsBackToOriginal(t *testing.T) {
	provider := &mockHyDEProvider{response: "  "}
	rewriter := NewHyDERewriter(provider)

	got, err := rewriter.Rewrite(context.Background(), "deployment steps")
	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	if got != "deployment steps" {
		t.Fatalf("Rewrite() = %q, want original query when LLM returns empty", got)
	}
}

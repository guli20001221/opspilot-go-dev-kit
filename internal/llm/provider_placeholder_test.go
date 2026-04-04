package llm

import (
	"context"
	"testing"
)

func TestPlaceholderProviderReturnsConstant(t *testing.T) {
	p := NewPlaceholderProvider()
	resp, err := p.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != PlaceholderContent {
		t.Fatalf("Content = %q, want %q", resp.Content, PlaceholderContent)
	}
	if resp.Model != "placeholder" {
		t.Fatalf("Model = %q, want %q", resp.Model, "placeholder")
	}
}

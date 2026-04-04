package retrieval

import (
	"context"
	"math"
	"testing"
)

func TestPlaceholderEmbedderDeterministic(t *testing.T) {
	e := &PlaceholderEmbedder{}
	v1, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	v2, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(v1) != EmbeddingDimension {
		t.Fatalf("len(v1) = %d, want %d", len(v1), EmbeddingDimension)
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("v1[%d] = %f, v2[%d] = %f — not deterministic", i, v1[i], i, v2[i])
		}
	}
}

func TestPlaceholderEmbedderUnitLength(t *testing.T) {
	e := &PlaceholderEmbedder{}
	v, err := e.Embed(context.Background(), "test vector")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	var norm float64
	for _, val := range v {
		norm += float64(val) * float64(val)
	}
	norm = math.Sqrt(norm)
	if math.Abs(norm-1.0) > 0.001 {
		t.Fatalf("norm = %f, want ~1.0", norm)
	}
}

func TestPlaceholderEmbedderDifferentTextDifferentVectors(t *testing.T) {
	e := &PlaceholderEmbedder{}
	v1, _ := e.Embed(context.Background(), "alpha")
	v2, _ := e.Embed(context.Background(), "beta")
	same := true
	for i := range v1 {
		if v1[i] != v2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different text produced identical vectors")
	}
}

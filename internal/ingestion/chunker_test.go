package ingestion

import (
	"context"
	"testing"
)

// distinctEmbedder returns different vectors per sentence to test boundary detection.
type distinctEmbedder struct {
	vectors map[string][]float32
}

func (e *distinctEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if vec, ok := e.vectors[text]; ok {
		return vec, nil
	}
	// Default: zero vector
	return make([]float32, 4), nil
}

func TestSemanticChunkerDetectsBoundaries(t *testing.T) {
	// Sentences 0-1 are similar (topic A), sentence 2 is different (topic B)
	embedder := &distinctEmbedder{
		vectors: map[string][]float32{
			"Sentence about passwords.":    {1, 0, 0, 0},
			"Reset your password here.":    {0.9, 0.1, 0, 0},
			"Refund policy is 30 days.":    {0, 0, 1, 0},
			"Return items within a month.": {0, 0, 0.9, 0.1},
		},
	}

	chunker := NewSemanticChunker(embedder, ChunkerOptions{
		Threshold:    0.5,
		MinSentences: 1,
		MaxSentences: 10,
	})

	sentences := []Sentence{
		{Text: "Sentence about passwords.", Index: 0},
		{Text: "Reset your password here.", Index: 1},
		{Text: "Refund policy is 30 days.", Index: 2},
		{Text: "Return items within a month.", Index: 3},
	}

	chunks, err := chunker.Chunk(context.Background(), sentences)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2 (split between password and refund topics)", len(chunks))
	}
	if len(chunks[0].Sentences) != 2 {
		t.Fatalf("chunks[0].Sentences = %d, want 2", len(chunks[0].Sentences))
	}
	if len(chunks[1].Sentences) != 2 {
		t.Fatalf("chunks[1].Sentences = %d, want 2", len(chunks[1].Sentences))
	}
}

func TestSemanticChunkerEnforcesMinSentences(t *testing.T) {
	embedder := &distinctEmbedder{
		vectors: map[string][]float32{
			"A": {1, 0, 0, 0},
			"B": {0, 1, 0, 0},
			"C": {0, 0, 1, 0},
		},
	}

	chunker := NewSemanticChunker(embedder, ChunkerOptions{
		Threshold:    0.5,
		MinSentences: 2, // Force at least 2 per chunk
		MaxSentences: 10,
	})

	sentences := []Sentence{
		{Text: "A", Index: 0},
		{Text: "B", Index: 1},
		{Text: "C", Index: 2},
	}

	chunks, err := chunker.Chunk(context.Background(), sentences)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// With min=2, 3 sentences with all-different embeddings should merge into at most 2 chunks
	// where the smallest group gets merged
	for _, c := range chunks {
		if len(c.Sentences) < 2 {
			t.Fatalf("chunk has %d sentences, want >= 2 (min constraint)", len(c.Sentences))
		}
	}
}

func TestSemanticChunkerSingleSentence(t *testing.T) {
	embedder := &distinctEmbedder{}
	chunker := NewSemanticChunker(embedder, ChunkerOptions{})

	chunks, err := chunker.Chunk(context.Background(), []Sentence{{Text: "Only one.", Index: 0}})
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
}

func TestSemanticChunkerEmbeddingsCarriedThrough(t *testing.T) {
	embedder := &distinctEmbedder{
		vectors: map[string][]float32{
			"A sentence.": {1, 0, 0, 0},
			"B sentence.": {0.95, 0.05, 0, 0},
		},
	}

	chunker := NewSemanticChunker(embedder, ChunkerOptions{MinSentences: 1, MaxSentences: 10})
	sentences := []Sentence{
		{Text: "A sentence.", Index: 0},
		{Text: "B sentence.", Index: 1},
	}

	chunks, err := chunker.Chunk(context.Background(), sentences)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// Verify embeddings are carried through to sentences
	for _, chunk := range chunks {
		for _, s := range chunk.Sentences {
			if len(s.Embedding) == 0 {
				t.Fatalf("sentence %q has no embedding", s.Text)
			}
		}
	}
}

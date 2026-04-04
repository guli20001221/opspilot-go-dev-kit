package ingestion

import (
	"context"
	"fmt"
	"math"
	"strings"

	"opspilot-go/internal/retrieval"
)

// ChunkerOptions configures the semantic chunker.
type ChunkerOptions struct {
	Threshold    float64 // similarity threshold for boundary detection; default 0.5
	MinSentences int     // minimum sentences per chunk; default 2
	MaxSentences int     // maximum sentences per chunk; default 15
}

// SemanticChunker groups sentences into semantically coherent chunks.
type SemanticChunker struct {
	embedder     retrieval.Embedder
	threshold    float64
	minSentences int
	maxSentences int
}

// NewSemanticChunker constructs the semantic chunker.
func NewSemanticChunker(embedder retrieval.Embedder, opts ChunkerOptions) *SemanticChunker {
	if opts.Threshold <= 0 {
		opts.Threshold = 0.5
	}
	if opts.MinSentences <= 0 {
		opts.MinSentences = 2
	}
	if opts.MaxSentences <= 0 {
		opts.MaxSentences = 15
	}
	return &SemanticChunker{
		embedder:     embedder,
		threshold:    opts.Threshold,
		minSentences: opts.MinSentences,
		maxSentences: opts.MaxSentences,
	}
}

// Chunk groups sentences into semantically coherent chunks.
func (c *SemanticChunker) Chunk(ctx context.Context, sentences []Sentence) ([]Chunk, error) {
	if len(sentences) == 0 {
		return nil, nil
	}
	if len(sentences) == 1 {
		return []Chunk{{
			ChunkID:   "chunk-0",
			Text:      sentences[0].Text,
			Sentences: sentences,
			IsParent:  true,
		}}, nil
	}

	// Embed all sentences
	embeddings := make([][]float32, len(sentences))
	for i, s := range sentences {
		vec, err := c.embedder.Embed(ctx, s.Text)
		if err != nil {
			return nil, fmt.Errorf("embed sentence %d: %w", i, err)
		}
		embeddings[i] = vec
	}

	// Compute consecutive similarities
	similarities := make([]float64, len(sentences)-1)
	for i := 0; i < len(similarities); i++ {
		similarities[i] = cosineSimilarity(embeddings[i], embeddings[i+1])
	}

	// Find boundaries where similarity drops below threshold
	var boundaries []int
	for i, sim := range similarities {
		if sim < c.threshold {
			boundaries = append(boundaries, i+1) // split BEFORE sentence i+1
		}
	}

	// Group sentences into chunks
	groups := splitAtBoundaries(sentences, boundaries)

	// Enforce min/max constraints
	groups = c.enforceConstraints(groups, similarities)

	// Build chunks
	chunks := make([]Chunk, 0, len(groups))
	for i, group := range groups {
		var texts []string
		for _, s := range group {
			texts = append(texts, s.Text)
		}
		chunks = append(chunks, Chunk{
			ChunkID:   fmt.Sprintf("chunk-%d", i),
			Text:      strings.Join(texts, " "),
			Sentences: group,
			IsParent:  true,
		})
	}

	return chunks, nil
}

func (c *SemanticChunker) enforceConstraints(groups [][]Sentence, similarities []float64) [][]Sentence {
	// Merge small groups into neighbors
	var merged [][]Sentence
	for _, group := range groups {
		if len(merged) > 0 && len(merged[len(merged)-1]) < c.minSentences {
			merged[len(merged)-1] = append(merged[len(merged)-1], group...)
		} else {
			merged = append(merged, group)
		}
	}
	// Handle trailing small group
	if len(merged) > 1 && len(merged[len(merged)-1]) < c.minSentences {
		merged[len(merged)-2] = append(merged[len(merged)-2], merged[len(merged)-1]...)
		merged = merged[:len(merged)-1]
	}

	// Split oversized groups
	var result [][]Sentence
	for _, group := range merged {
		if len(group) <= c.maxSentences {
			result = append(result, group)
			continue
		}
		// Split at midpoint
		mid := len(group) / 2
		result = append(result, group[:mid], group[mid:])
	}

	return result
}

func splitAtBoundaries(sentences []Sentence, boundaries []int) [][]Sentence {
	if len(boundaries) == 0 {
		return [][]Sentence{sentences}
	}

	var groups [][]Sentence
	prev := 0
	for _, b := range boundaries {
		if b > prev {
			groups = append(groups, sentences[prev:b])
		}
		prev = b
	}
	if prev < len(sentences) {
		groups = append(groups, sentences[prev:])
	}
	return groups
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

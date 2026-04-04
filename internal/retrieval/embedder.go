package retrieval

import (
	"context"
	"crypto/sha256"
	"math"
)

// EmbeddingDimension is the vector dimension used by the retrieval system.
const EmbeddingDimension = 1536

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// PlaceholderEmbedder produces deterministic hash-based embeddings.
// Same text always produces the same unit vector.
type PlaceholderEmbedder struct{}

// Embed generates a deterministic unit vector from the SHA-256 hash of the input text.
func (e *PlaceholderEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	hash := sha256.Sum256([]byte(text))
	vec := make([]float32, EmbeddingDimension)

	// Cycle hash bytes across the vector dimensions
	for i := range vec {
		byteIdx := i % len(hash)
		vec[i] = float32(hash[byteIdx]) / 255.0
	}

	// Normalize to unit length
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}

	return vec, nil
}

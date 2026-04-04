package retrieval

import (
	"context"
	"crypto/sha256"
	"math"
)

// DefaultEmbeddingDimension is the default vector dimension for the placeholder embedder.
const DefaultEmbeddingDimension = 4096

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// PlaceholderEmbedder produces deterministic hash-based embeddings.
// Same text always produces the same unit vector.
type PlaceholderEmbedder struct {
	Dimension int
}

// Embed generates a deterministic unit vector from the SHA-256 hash of the input text.
func (e *PlaceholderEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	dim := e.Dimension
	if dim <= 0 {
		dim = DefaultEmbeddingDimension
	}
	hash := sha256.Sum256([]byte(text))
	vec := make([]float32, dim)

	for i := range vec {
		byteIdx := i % len(hash)
		vec[i] = float32(hash[byteIdx]) / 255.0
	}

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

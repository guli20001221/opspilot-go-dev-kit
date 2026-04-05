package contextengine

import (
	"context"
	"log/slog"
	"math"
	"strings"

	"opspilot-go/internal/retrieval"
)

// EmbeddingImportanceScorer uses embedding cosine similarity to dynamically
// score block importance relative to the current query. Blocks whose content
// is semantically closer to the query get higher priority, while less relevant
// blocks get lower priority and are more likely to be evicted under budget
// pressure.
//
// This implements the MemGPT insight: the context window is a limited resource
// that the system actively manages based on the current task, rather than
// filling it with a fixed priority ordering.
type EmbeddingImportanceScorer struct {
	embedder retrieval.Embedder
	// basePriority maps block kinds to their minimum priority floor.
	// Dynamic scoring adjusts within [floor, floor+40] range.
	basePriority map[string]int
}

// NewEmbeddingImportanceScorer constructs an importance scorer using embedding similarity.
// Returns nil if the embedder is nil or a placeholder.
func NewEmbeddingImportanceScorer(embedder retrieval.Embedder) *EmbeddingImportanceScorer {
	if embedder == nil {
		return nil
	}
	return &EmbeddingImportanceScorer{
		embedder: embedder,
		basePriority: map[string]int{
			BlockKindUserProfile:        80, // always high (identity context)
			BlockKindRecentTurns:        70, // conversation continuity
			BlockKindRetrievalEvidence:  50, // scored by content relevance
			BlockKindToolResult:         50, // scored by content relevance
			BlockKindSessionSummary:     30, // background context
			BlockKindTaskScratchpad:     30, // task notes
		},
	}
}

// ScoreBlocks computes embedding similarity between the query and each block's
// content, then maps the similarity score to a priority value within the block
// kind's allowed range [basePriority, basePriority+40].
func (s *EmbeddingImportanceScorer) ScoreBlocks(ctx context.Context, query string, blocks []Block) {
	if s == nil || len(blocks) == 0 || query == "" {
		return
	}

	queryEmb, err := s.embedder.Embed(ctx, query)
	if err != nil {
		slog.Warn("importance scorer: failed to embed query, keeping static priorities",
			slog.Any("error", err))
		return
	}

	for i := range blocks {
		blockEmb, embErr := s.embedder.Embed(ctx, blocks[i].Content)
		if embErr != nil {
			continue // keep static priority
		}

		similarity := cosineSimilarity(queryEmb, blockEmb)
		// Clamp to [0,1] — negative similarity (anti-correlated) maps to floor
		if similarity < 0 {
			similarity = 0
		}
		if similarity > 1 {
			similarity = 1
		}

		base := s.basePriority[blocks[i].Kind]
		if base == 0 {
			base = 40 // default floor for unknown block kinds
		}

		// Map similarity [0,1] → priority [base, base+40]
		dynamicPriority := base + int(similarity*40)
		blocks[i].Priority = dynamicPriority
		blocks[i].IncludeReason = blocks[i].IncludeReason + " (dynamic priority)"
	}
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector is zero-length.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
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

// KeywordImportanceScorer is a lightweight scorer that boosts blocks containing
// query keywords. No embeddings required — suitable for fallback when no
// embedder is available.
type KeywordImportanceScorer struct{}

// ScoreBlocks boosts priority by 20 for blocks that contain any word from the query.
func (KeywordImportanceScorer) ScoreBlocks(_ context.Context, query string, blocks []Block) {
	if query == "" || len(blocks) == 0 {
		return
	}

	words := splitWords(query)
	if len(words) == 0 {
		return
	}

	for i := range blocks {
		matchCount := 0
		for _, word := range words {
			if containsWord(blocks[i].Content, word) {
				matchCount++
			}
		}
		if matchCount > 0 {
			boost := 10 + (matchCount * 5)
			if boost > 30 {
				boost = 30
			}
			blocks[i].Priority += boost
			blocks[i].IncludeReason = blocks[i].IncludeReason + " (keyword boost)"
		}
	}
}

func splitWords(s string) []string {
	var words []string
	word := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			if len(word) >= 3 { // skip short words
				words = append(words, word)
			}
			word = ""
		} else {
			word += string(ch)
		}
	}
	if len(word) >= 3 {
		words = append(words, word)
	}
	return words
}

func containsWord(content, word string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(word))
}

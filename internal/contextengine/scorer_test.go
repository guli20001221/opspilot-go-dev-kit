package contextengine

import (
	"context"
	"math"
	"testing"
)

// --- Keyword scorer tests ---

func TestKeywordImportanceScorerBoostsMatchingBlocks(t *testing.T) {
	blocks := []Block{
		{Kind: BlockKindRecentTurns, Content: "user: How do I reset my password?", Priority: 90},
		{Kind: BlockKindSessionSummary, Content: "User discussed account settings earlier.", Priority: 50},
		{Kind: BlockKindTaskScratchpad, Content: "Unrelated task notes about deployment.", Priority: 40},
	}

	scorer := KeywordImportanceScorer{}
	scorer.ScoreBlocks(context.Background(), "reset password", blocks)

	// "reset" and "password" match the first block → boosted
	if blocks[0].Priority <= 90 {
		t.Fatalf("recent turns Priority = %d, want > 90 (keyword match)", blocks[0].Priority)
	}
	// "deployment" doesn't match → unchanged
	if blocks[2].Priority != 40 {
		t.Fatalf("scratchpad Priority = %d, want 40 (no match)", blocks[2].Priority)
	}
}

func TestKeywordImportanceScorerCapsBoost(t *testing.T) {
	blocks := []Block{
		{Kind: BlockKindRecentTurns, Content: "reset password security settings authentication two-factor recovery", Priority: 50},
	}

	scorer := KeywordImportanceScorer{}
	scorer.ScoreBlocks(context.Background(), "reset password security settings authentication recovery", blocks)

	// Many matches but boost is capped at 30
	if blocks[0].Priority > 80 {
		t.Fatalf("Priority = %d, want <= 80 (50 base + 30 cap)", blocks[0].Priority)
	}
}

func TestKeywordImportanceScorerNoOpOnEmptyQuery(t *testing.T) {
	blocks := []Block{
		{Kind: BlockKindRecentTurns, Content: "some content", Priority: 90},
	}
	scorer := KeywordImportanceScorer{}
	scorer.ScoreBlocks(context.Background(), "", blocks)
	if blocks[0].Priority != 90 {
		t.Fatalf("Priority = %d, want 90 (unchanged on empty query)", blocks[0].Priority)
	}
}

// --- Cosine similarity tests ---

func TestCosineSimilarityIdenticalVectors(t *testing.T) {
	v := []float32{1, 2, 3, 4}
	sim := cosineSimilarity(v, v)
	if math.Abs(sim-1.0) > 0.001 {
		t.Fatalf("similarity = %f, want ~1.0 for identical vectors", sim)
	}
}

func TestCosineSimilarityOrthogonalVectors(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim) > 0.001 {
		t.Fatalf("similarity = %f, want ~0.0 for orthogonal vectors", sim)
	}
}

func TestCosineSimilarityEmptyVectors(t *testing.T) {
	sim := cosineSimilarity(nil, nil)
	if sim != 0 {
		t.Fatalf("similarity = %f, want 0 for empty vectors", sim)
	}
}

func TestCosineSimilarityDifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Fatalf("similarity = %f, want 0 for different-length vectors", sim)
	}
}

// --- Embedding scorer tests ---

type mockEmbedder struct {
	vectors map[string][]float32
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if v, ok := m.vectors[text]; ok {
		return v, nil
	}
	// Default: return a generic embedding
	return []float32{0.1, 0.1, 0.1, 0.1}, nil
}

func TestEmbeddingImportanceScorerAdjustsPriorities(t *testing.T) {
	embedder := &mockEmbedder{
		vectors: map[string][]float32{
			"reset password":                        {0.9, 0.1, 0.0, 0.0},
			"user: How do I reset my password?":     {0.8, 0.2, 0.0, 0.0},         // similar to query
			"Unrelated notes about server deployment": {0.0, 0.0, 0.9, 0.1},       // dissimilar
		},
	}

	scorer := NewEmbeddingImportanceScorer(embedder)
	blocks := []Block{
		{Kind: BlockKindRecentTurns, Content: "user: How do I reset my password?", Priority: 70},
		{Kind: BlockKindTaskScratchpad, Content: "Unrelated notes about server deployment", Priority: 30},
	}

	scorer.ScoreBlocks(context.Background(), "reset password", blocks)

	// First block should get higher priority (similar to query)
	// Second block should get lower priority (dissimilar)
	if blocks[0].Priority <= blocks[1].Priority {
		t.Fatalf("relevant block Priority (%d) <= irrelevant block Priority (%d), want relevant > irrelevant",
			blocks[0].Priority, blocks[1].Priority)
	}
}

func TestEmbeddingImportanceScorerNilSafe(t *testing.T) {
	var scorer *EmbeddingImportanceScorer
	blocks := []Block{{Kind: BlockKindRecentTurns, Content: "test", Priority: 90}}
	scorer.ScoreBlocks(context.Background(), "query", blocks)
	if blocks[0].Priority != 90 {
		t.Fatal("nil scorer should not modify priorities")
	}
}

func TestEmbeddingImportanceScorerNilEmbedderReturnsNil(t *testing.T) {
	scorer := NewEmbeddingImportanceScorer(nil)
	if scorer != nil {
		t.Fatal("nil embedder should return nil scorer")
	}
}

// --- Integration: scorer with Build ---

func TestServiceBuildWithDynamicImportanceScoring(t *testing.T) {
	scorer := KeywordImportanceScorer{}
	svc := NewServiceWithDependencies(Config{Budget: 4096}, nil, scorer)

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID:   "req-scoring",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "password reset",
		RecentTurns: []Turn{
			{Role: "user", Content: "How do I reset my password?"},
		},
		TaskScratchpad: "Deployment notes for production server.",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Recent turns should have boosted priority (contains "password" and "reset")
	// Task scratchpad should be unchanged (no keyword match)
	var turnsPriority, scratchpadPriority int
	for _, block := range got.Planner.Blocks {
		switch block.Kind {
		case BlockKindRecentTurns:
			turnsPriority = block.Priority
		case BlockKindTaskScratchpad:
			scratchpadPriority = block.Priority
		}
	}
	if turnsPriority <= scratchpadPriority {
		t.Fatalf("turns Priority (%d) <= scratchpad Priority (%d), want turns boosted by keyword match",
			turnsPriority, scratchpadPriority)
	}
}

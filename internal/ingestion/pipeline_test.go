package ingestion

import (
	"context"
	"sync"
	"testing"

	"opspilot-go/internal/llm"
	"opspilot-go/internal/retrieval"
)

type memoryChunkStore struct {
	mu      sync.Mutex
	records []ChunkRecord
}

func (s *memoryChunkStore) UpsertWithHybrid(_ context.Context, chunk ChunkRecord) (ChunkRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, chunk)
	return chunk, nil
}

func (s *memoryChunkStore) DeleteStaleChunks(_ context.Context, _, _ string, _ int) (int, error) {
	return 0, nil
}

func TestPipelineEndToEnd(t *testing.T) {
	store := &memoryChunkStore{}
	embedder := &retrieval.PlaceholderEmbedder{}
	provider := llm.NewPlaceholderProvider()

	pipeline := NewPipeline(embedder, provider, store, PipelineOptions{})

	doc := Document{
		DocumentID:      "doc-test-1",
		TenantID:        "tenant-test",
		DocumentVersion: 1,
		SourceTitle:     "Password Reset Guide",
		SourceURI:       "https://docs.example.com/password-reset",
		Content: `How to reset your password. Navigate to Settings and find Security. Click Reset Password and follow the prompts.

Account recovery options. If you cannot access your email, contact support. Provide your account ID and verification.

Two-factor authentication setup. Enable 2FA in Security settings. Use an authenticator app for best security.`,
	}

	result, err := pipeline.Ingest(context.Background(), doc)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.DocumentID != "doc-test-1" {
		t.Fatalf("DocumentID = %q, want %q", result.DocumentID, "doc-test-1")
	}
	if result.ParentChunks == 0 {
		t.Fatal("ParentChunks = 0, want > 0")
	}
	if result.ChildChunks == 0 {
		t.Fatal("ChildChunks = 0, want > 0")
	}
	if result.ChunksStored != result.ParentChunks+result.ChildChunks {
		t.Fatalf("ChunksStored = %d, want %d", result.ChunksStored, result.ParentChunks+result.ChildChunks)
	}

	// Verify all records have embeddings
	for _, rec := range store.records {
		if len(rec.Embedding) == 0 {
			t.Fatalf("chunk %q has no embedding", rec.ChunkID)
		}
	}

	// Verify parent-child structure
	parents := 0
	children := 0
	for _, rec := range store.records {
		if rec.ParentChunkID == nil {
			parents++
		} else {
			children++
		}
	}
	if parents != result.ParentChunks {
		t.Fatalf("parent records = %d, want %d", parents, result.ParentChunks)
	}
	if children != result.ChildChunks {
		t.Fatalf("child records = %d, want %d", children, result.ChildChunks)
	}

	// Verify context prefixes populated
	for _, rec := range store.records {
		if rec.ContextPrefix == "" {
			t.Fatalf("chunk %q has empty context prefix", rec.ChunkID)
		}
	}

	// Verify idempotency — re-ingest produces same chunk IDs
	store2 := &memoryChunkStore{}
	pipeline2 := NewPipeline(embedder, provider, store2, PipelineOptions{})
	result2, err := pipeline2.Ingest(context.Background(), doc)
	if err != nil {
		t.Fatalf("re-Ingest() error = %v", err)
	}
	if result2.ChunksStored != result.ChunksStored {
		t.Fatalf("re-ingest ChunksStored = %d, want %d", result2.ChunksStored, result.ChunksStored)
	}
	for i := range store.records {
		if store.records[i].ChunkID != store2.records[i].ChunkID {
			t.Fatalf("chunk ID mismatch at %d: %q vs %q", i, store.records[i].ChunkID, store2.records[i].ChunkID)
		}
	}
}

func TestPipelineRejectsEmptyDocument(t *testing.T) {
	pipeline := NewPipeline(&retrieval.PlaceholderEmbedder{}, llm.NewPlaceholderProvider(), &memoryChunkStore{}, PipelineOptions{})
	_, err := pipeline.Ingest(context.Background(), Document{})
	if err == nil {
		t.Fatal("Ingest() error = nil, want validation error")
	}
}

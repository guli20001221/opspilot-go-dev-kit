package postgres

import (
	"context"
	"os"
	"testing"

	"opspilot-go/internal/retrieval"
)

func TestRetrievalChunkStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DELETE FROM retrieval_chunks"); err != nil {
		t.Fatalf("DELETE retrieval_chunks error = %v", err)
	}

	embedder := &retrieval.PlaceholderEmbedder{}
	store := NewRetrievalChunkStore(pool, embedder)

	// Generate embedding for upsert
	embedding, err := embedder.Embed(ctx, "How do I reset my password?")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	chunk := RetrievalChunk{
		ID:               "chunk-pg-test-1",
		TenantID:         "tenant-retrieval",
		DocumentID:       "doc-1",
		DocumentVersion:  1,
		ChunkID:          "chunk-1",
		SourceTitle:      "Password Reset Guide",
		SourceURI:        "https://docs.example.com/password-reset",
		Snippet:          "To reset your password, navigate to Settings > Security > Reset Password.",
		Embedding:        embedding,
		PermissionsScope: "tenant-retrieval",
	}

	// Upsert
	saved, err := store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if saved.ID != chunk.ID {
		t.Fatalf("saved.ID = %q, want %q", saved.ID, chunk.ID)
	}
	if saved.SourceTitle != "Password Reset Guide" {
		t.Fatalf("saved.SourceTitle = %q, want %q", saved.SourceTitle, "Password Reset Guide")
	}

	// Search — same text should return the chunk with high similarity
	result, err := store.Search(ctx, retrieval.RetrievalRequest{
		RequestID: "req-test-1",
		TenantID:  "tenant-retrieval",
		QueryText: "How do I reset my password?",
		TopK:      5,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(result.EvidenceBlocks) != 1 {
		t.Fatalf("len(EvidenceBlocks) = %d, want 1", len(result.EvidenceBlocks))
	}
	if result.EvidenceBlocks[0].EvidenceID != "chunk-pg-test-1" {
		t.Fatalf("EvidenceBlocks[0].EvidenceID = %q, want %q", result.EvidenceBlocks[0].EvidenceID, "chunk-pg-test-1")
	}
	if result.EvidenceBlocks[0].Score <= 0 {
		t.Fatalf("EvidenceBlocks[0].Score = %f, want > 0", result.EvidenceBlocks[0].Score)
	}

	// Tenant isolation — different tenant returns no results
	otherResult, err := store.Search(ctx, retrieval.RetrievalRequest{
		RequestID: "req-test-2",
		TenantID:  "tenant-other",
		QueryText: "password reset",
		TopK:      5,
	})
	if err != nil {
		t.Fatalf("Search(other tenant) error = %v", err)
	}
	if len(otherResult.EvidenceBlocks) != 0 {
		t.Fatalf("len(EvidenceBlocks) for other tenant = %d, want 0", len(otherResult.EvidenceBlocks))
	}

	// Upsert dedup — same (document_id, chunk_id) updates instead of inserting
	chunk.Snippet = "Updated snippet for password reset."
	embedding2, _ := embedder.Embed(ctx, "Updated password reset content")
	chunk.Embedding = embedding2
	updated, err := store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("Upsert(update) error = %v", err)
	}
	if updated.Snippet != "Updated snippet for password reset." {
		t.Fatalf("updated.Snippet = %q, want updated text", updated.Snippet)
	}
}

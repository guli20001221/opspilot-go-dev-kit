package ingestion

import (
	"context"
	"time"
)

// Document is the input to the ingestion pipeline.
type Document struct {
	DocumentID       string
	TenantID         string
	DocumentVersion  int
	SourceTitle      string
	SourceURI        string
	Content          string
	PermissionsScope string
	PublishedAt      *time.Time
}

// Sentence is a single sentence extracted from a document.
type Sentence struct {
	Text      string
	Index     int // ordinal position (0-based)
	Embedding []float32
}

// Chunk is a semantically coherent group of sentences.
type Chunk struct {
	ChunkID       string
	Sentences     []Sentence
	Text          string
	ContextPrefix string
	IsParent      bool
}

// IngestResult is the typed output of one document ingestion.
type IngestResult struct {
	DocumentID   string
	TenantID     string
	ChunksStored int
	ParentChunks int
	ChildChunks  int
}

// ChunkRecord is the storage-layer representation for upsert.
type ChunkRecord struct {
	ID               string
	TenantID         string
	DocumentID       string
	DocumentVersion  int
	ChunkID          string
	ParentChunkID    *string
	SourceTitle      string
	SourceURI        string
	Snippet          string
	ContextPrefix    string
	Embedding        []float32
	PermissionsScope string
	PublishedAt      *time.Time
}

// ChunkStore persists retrieval chunks with hybrid index data.
type ChunkStore interface {
	UpsertWithHybrid(ctx context.Context, chunk ChunkRecord) (ChunkRecord, error)
	// DeleteStaleChunks removes chunks for the given document that are stale.
	// It deletes chunks with a version older than currentVersion, AND chunks
	// with currentVersion whose chunk_id is not in currentChunkIDs. This
	// handles both cross-version and same-version re-ingestion (where the new
	// chunking produces fewer chunks than before).
	DeleteStaleChunks(ctx context.Context, tenantID, documentID string, currentVersion int, currentChunkIDs []string) (int, error)
}

package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/ingestion"
	"opspilot-go/internal/retrieval"
)

// RetrievalChunkStore persists and queries retrieval chunks with pgvector.
type RetrievalChunkStore struct {
	pool     *pgxpool.Pool
	embedder retrieval.Embedder
}

// NewRetrievalChunkStore constructs the retrieval chunk repository.
func NewRetrievalChunkStore(pool *pgxpool.Pool, embedder retrieval.Embedder) *RetrievalChunkStore {
	return &RetrievalChunkStore{pool: pool, embedder: embedder}
}

// RetrievalChunk is the storage representation of a retrieval chunk for upsert.
type RetrievalChunk struct {
	ID               string
	TenantID         string
	DocumentID       string
	DocumentVersion  int
	ChunkID          string
	SourceTitle      string
	SourceURI        string
	Snippet          string
	Embedding        []float32
	PermissionsScope string
	PublishedAt      *time.Time
}

// Upsert inserts or updates a retrieval chunk row.
func (s *RetrievalChunkStore) Upsert(ctx context.Context, chunk RetrievalChunk) (RetrievalChunk, error) {
	const query = `
INSERT INTO retrieval_chunks (
    id, tenant_id, document_id, document_version, chunk_id,
    source_title, source_uri, snippet, embedding,
    permissions_scope, published_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9::vector, $10, $11
)
ON CONFLICT (document_id, chunk_id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    document_version = EXCLUDED.document_version,
    source_title = EXCLUDED.source_title,
    source_uri = EXCLUDED.source_uri,
    snippet = EXCLUDED.snippet,
    embedding = EXCLUDED.embedding,
    permissions_scope = EXCLUDED.permissions_scope,
    published_at = EXCLUDED.published_at
RETURNING id, tenant_id, document_id, document_version, chunk_id,
          source_title, source_uri, snippet, permissions_scope, published_at`

	var out RetrievalChunk
	err := s.pool.QueryRow(ctx, query,
		chunk.ID, chunk.TenantID, chunk.DocumentID, chunk.DocumentVersion, chunk.ChunkID,
		chunk.SourceTitle, chunk.SourceURI, chunk.Snippet, formatVector(chunk.Embedding),
		chunk.PermissionsScope, chunk.PublishedAt,
	).Scan(&out.ID, &out.TenantID, &out.DocumentID, &out.DocumentVersion, &out.ChunkID,
		&out.SourceTitle, &out.SourceURI, &out.Snippet, &out.PermissionsScope, &out.PublishedAt)
	if err != nil {
		return RetrievalChunk{}, fmt.Errorf("upsert retrieval chunk: %w", err)
	}
	return out, nil
}

// UpsertWithHybrid inserts or updates a chunk with parent-child linking and contextual prefix.
// The search_tsv column is auto-computed by a database trigger.
func (s *RetrievalChunkStore) UpsertWithHybrid(ctx context.Context, chunk ingestion.ChunkRecord) (ingestion.ChunkRecord, error) {
	const query = `
INSERT INTO retrieval_chunks (
    id, tenant_id, document_id, document_version, chunk_id,
    parent_chunk_id, source_title, source_uri, snippet, context_prefix,
    embedding, permissions_scope, published_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::vector, $12, $13
)
ON CONFLICT (document_id, chunk_id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    document_version = EXCLUDED.document_version,
    parent_chunk_id = EXCLUDED.parent_chunk_id,
    source_title = EXCLUDED.source_title,
    source_uri = EXCLUDED.source_uri,
    snippet = EXCLUDED.snippet,
    context_prefix = EXCLUDED.context_prefix,
    embedding = EXCLUDED.embedding,
    permissions_scope = EXCLUDED.permissions_scope,
    published_at = EXCLUDED.published_at
RETURNING id, tenant_id, document_id, document_version, chunk_id,
          parent_chunk_id, source_title, source_uri, snippet, context_prefix,
          permissions_scope, published_at`

	var out ingestion.ChunkRecord
	err := s.pool.QueryRow(ctx, query,
		chunk.ID, chunk.TenantID, chunk.DocumentID, chunk.DocumentVersion, chunk.ChunkID,
		chunk.ParentChunkID, chunk.SourceTitle, chunk.SourceURI, chunk.Snippet, chunk.ContextPrefix,
		formatVector(chunk.Embedding), chunk.PermissionsScope, chunk.PublishedAt,
	).Scan(&out.ID, &out.TenantID, &out.DocumentID, &out.DocumentVersion, &out.ChunkID,
		&out.ParentChunkID, &out.SourceTitle, &out.SourceURI, &out.Snippet, &out.ContextPrefix,
		&out.PermissionsScope, &out.PublishedAt)
	if err != nil {
		return ingestion.ChunkRecord{}, fmt.Errorf("upsert hybrid chunk: %w", err)
	}
	return out, nil
}

// Verify that RetrievalChunkStore implements ingestion.ChunkStore.
var _ ingestion.ChunkStore = (*RetrievalChunkStore)(nil)

// Search embeds the query text and returns the top-K most similar chunks for the tenant.
func (s *RetrievalChunkStore) Search(ctx context.Context, req retrieval.RetrievalRequest) (retrieval.RetrievalResult, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}

	queryVec, err := s.embedder.Embed(ctx, req.QueryText)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("embed query text: %w", err)
	}

	const query = `
SELECT id, tenant_id, document_id, document_version, chunk_id,
       source_title, source_uri, snippet, permissions_scope, published_at,
       1 - (embedding <=> $1::vector) AS score
FROM retrieval_chunks
WHERE tenant_id = $2
ORDER BY embedding <=> $1::vector
LIMIT $3`

	rows, err := s.pool.Query(ctx, query, formatVector(queryVec), req.TenantID, topK)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	var blocks []retrieval.EvidenceBlock
	for rows.Next() {
		var block retrieval.EvidenceBlock
		if err := rows.Scan(
			&block.EvidenceID, &block.TenantID, &block.DocumentID, &block.DocumentVersion,
			&block.ChunkID, &block.SourceTitle, &block.SourceURI, &block.Snippet,
			&block.PermissionsScope, &block.PublishedAt, &block.Score,
		); err != nil {
			return retrieval.RetrievalResult{}, fmt.Errorf("scan retrieval chunk: %w", err)
		}
		block.CitationLabel = fmt.Sprintf("[%d]", len(blocks)+1)
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("iterate retrieval chunks: %w", err)
	}

	var coverage float64
	if len(blocks) > 0 {
		var totalScore float64
		for _, b := range blocks {
			totalScore += b.Score
		}
		coverage = totalScore / float64(len(blocks))
	}

	return retrieval.RetrievalResult{
		RequestID:      req.RequestID,
		PlanID:         req.PlanID,
		QueryUsed:      req.QueryText,
		EvidenceBlocks: blocks,
		CoverageScore:  coverage,
	}, nil
}

// formatVector converts a float32 slice to pgvector text format.
func formatVector(vec []float32) string {
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// Verify that RetrievalChunkStore implements retrieval.Searcher.
var _ retrieval.Searcher = (*RetrievalChunkStore)(nil)

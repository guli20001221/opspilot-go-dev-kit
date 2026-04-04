package postgres

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"opspilot-go/internal/observability/tracing"

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

// DeleteStaleChunks removes chunks from older document versions.
func (s *RetrievalChunkStore) DeleteStaleChunks(ctx context.Context, tenantID, documentID string, currentVersion int) (int, error) {
	const query = `
DELETE FROM retrieval_chunks
WHERE tenant_id = $1 AND document_id = $2 AND document_version < $3`

	tag, err := s.pool.Exec(ctx, query, tenantID, documentID, currentVersion)
	if err != nil {
		return 0, fmt.Errorf("delete stale chunks: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// Verify that RetrievalChunkStore implements ingestion.ChunkStore.
var _ ingestion.ChunkStore = (*RetrievalChunkStore)(nil)

// rrfK is the constant used in Reciprocal Rank Fusion: score = 1/(k + rank).
const rrfK = 60

// Search performs hybrid retrieval: dense vector search + BM25 full-text search,
// fused with Reciprocal Rank Fusion (RRF). Child chunks are expanded to their
// parent chunks for richer LLM context.
func (s *RetrievalChunkStore) Search(ctx context.Context, req retrieval.RetrievalRequest) (retrieval.RetrievalResult, error) {
	queryPreview := req.QueryText
	if len(queryPreview) > 128 {
		queryPreview = queryPreview[:128]
	}
	ctx, span := tracing.StartSpan(ctx, "retrieval.search",
		tracing.AttrTenantID.String(req.TenantID),
	)
	defer span.End()
	_ = queryPreview

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	// Fetch more candidates from each source for better fusion
	candidateK := topK * 3
	if candidateK < 20 {
		candidateK = 20
	}

	// Dense search uses RewrittenQuery (HyDE output) when available for
	// better semantic matching; BM25 uses original QueryText for keyword precision.
	denseQueryText := req.QueryText
	if req.RewrittenQuery != "" {
		denseQueryText = req.RewrittenQuery
	}

	queryVec, err := s.embedder.Embed(ctx, denseQueryText)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("embed query text: %w", err)
	}

	// Stage 1: Dense vector search (uses HyDE-rewritten query when available)
	denseBlocks, err := s.denseSearch(ctx, queryVec, req.TenantID, candidateK)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("dense search: %w", err)
	}

	// Stage 2: BM25 full-text search (uses original query for keyword precision)
	bm25Blocks, err := s.bm25Search(ctx, req.QueryText, req.TenantID, candidateK)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("bm25 search: %w", err)
	}

	// Stage 3: Reciprocal Rank Fusion
	fused := reciprocalRankFusion(denseBlocks, bm25Blocks, topK)

	// Stage 4: Parent-child expansion (retrieve child → serve parent for richer context)
	expanded, err := s.expandToParents(ctx, fused, req.TenantID)
	if err != nil {
		return retrieval.RetrievalResult{}, fmt.Errorf("parent expansion: %w", err)
	}

	// Assign citation labels and compute coverage
	for i := range expanded {
		expanded[i].CitationLabel = fmt.Sprintf("[%d]", i+1)
	}

	var coverage float64
	if len(expanded) > 0 {
		var totalScore float64
		for _, b := range expanded {
			totalScore += b.Score
		}
		coverage = totalScore / float64(len(expanded))
	}

	return retrieval.RetrievalResult{
		RequestID:      req.RequestID,
		PlanID:         req.PlanID,
		QueryUsed:      req.QueryText,
		EvidenceBlocks: expanded,
		CoverageScore:  coverage,
	}, nil
}

func (s *RetrievalChunkStore) denseSearch(ctx context.Context, queryVec []float32, tenantID string, limit int) ([]retrieval.EvidenceBlock, error) {
	const query = `
SELECT id, tenant_id, document_id, document_version, chunk_id,
       source_title, source_uri, snippet, permissions_scope, published_at,
       parent_chunk_id,
       1 - (embedding <=> $1::vector) AS score
FROM retrieval_chunks
WHERE tenant_id = $2
ORDER BY embedding <=> $1::vector
LIMIT $3`

	return s.scanBlocks(ctx, query, formatVector(queryVec), tenantID, limit)
}

func (s *RetrievalChunkStore) bm25Search(ctx context.Context, queryText, tenantID string, limit int) ([]retrieval.EvidenceBlock, error) {
	const query = `
SELECT id, tenant_id, document_id, document_version, chunk_id,
       source_title, source_uri, snippet, permissions_scope, published_at,
       parent_chunk_id,
       ts_rank_cd(search_tsv, plainto_tsquery('english', $1)) AS score
FROM retrieval_chunks
WHERE tenant_id = $2
  AND search_tsv @@ plainto_tsquery('english', $1)
ORDER BY ts_rank_cd(search_tsv, plainto_tsquery('english', $1)) DESC
LIMIT $3`

	return s.scanBlocks(ctx, query, queryText, tenantID, limit)
}

func (s *RetrievalChunkStore) scanBlocks(ctx context.Context, query string, param1 any, tenantID string, limit int) ([]retrieval.EvidenceBlock, error) {
	rows, err := s.pool.Query(ctx, query, param1, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []retrieval.EvidenceBlock
	for rows.Next() {
		var block retrieval.EvidenceBlock
		var parentChunkID *string
		if err := rows.Scan(
			&block.EvidenceID, &block.TenantID, &block.DocumentID, &block.DocumentVersion,
			&block.ChunkID, &block.SourceTitle, &block.SourceURI, &block.Snippet,
			&block.PermissionsScope, &block.PublishedAt, &parentChunkID, &block.Score,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

// reciprocalRankFusion merges two ranked lists using RRF: score = sum(1/(k+rank)) across lists.
func reciprocalRankFusion(listA, listB []retrieval.EvidenceBlock, topK int) []retrieval.EvidenceBlock {
	scores := make(map[string]float64)
	blockByID := make(map[string]retrieval.EvidenceBlock)

	for rank, b := range listA {
		scores[b.EvidenceID] += 1.0 / float64(rrfK+rank+1)
		blockByID[b.EvidenceID] = b
	}
	for rank, b := range listB {
		scores[b.EvidenceID] += 1.0 / float64(rrfK+rank+1)
		if _, exists := blockByID[b.EvidenceID]; !exists {
			blockByID[b.EvidenceID] = b
		}
	}

	type scored struct {
		id    string
		score float64
	}
	var ranked []scored
	for id, score := range scores {
		ranked = append(ranked, scored{id, score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].id < ranked[j].id // deterministic tiebreaker
	})

	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	// Normalize RRF scores to [0,1] range. Theoretical max is 2/(rrfK+1) when
	// a block appears at rank 1 in both lists.
	maxRRF := 2.0 / float64(rrfK+1)
	result := make([]retrieval.EvidenceBlock, 0, len(ranked))
	for _, r := range ranked {
		block := blockByID[r.id]
		normalized := r.score / maxRRF
		if normalized > 1.0 {
			normalized = 1.0
		}
		block.Score = normalized
		block.RerankScore = r.score
		result = append(result, block)
	}
	return result
}

// expandToParents replaces child chunks with their parent chunks for richer LLM context.
// Deduplicates parents so the same parent isn't returned multiple times.
func (s *RetrievalChunkStore) expandToParents(ctx context.Context, blocks []retrieval.EvidenceBlock, tenantID string) ([]retrieval.EvidenceBlock, error) {
	if len(blocks) == 0 {
		return blocks, nil
	}

	// Collect chunk IDs to look up parent mappings
	chunkIDs := make([]string, 0, len(blocks))
	for _, b := range blocks {
		chunkIDs = append(chunkIDs, b.ChunkID)
	}

	// Query parent_chunk_id for each block
	const parentQuery = `
SELECT chunk_id, parent_chunk_id
FROM retrieval_chunks
WHERE tenant_id = $1 AND chunk_id = ANY($2) AND parent_chunk_id IS NOT NULL`

	rows, err := s.pool.Query(ctx, parentQuery, tenantID, chunkIDs)
	if err != nil {
		return nil, fmt.Errorf("query parent mappings: %w", err)
	}
	defer rows.Close()

	childToParent := make(map[string]string)
	var parentIDs []string
	parentSeen := make(map[string]bool)
	for rows.Next() {
		var chunkID, parentID string
		if err := rows.Scan(&chunkID, &parentID); err != nil {
			return nil, err
		}
		childToParent[chunkID] = parentID
		if !parentSeen[parentID] {
			parentSeen[parentID] = true
			parentIDs = append(parentIDs, parentID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(parentIDs) == 0 {
		// No parent expansion needed (all blocks are already parents)
		return blocks, nil
	}

	// Fetch parent chunk details
	const fetchQuery = `
SELECT id, tenant_id, document_id, document_version, chunk_id,
       source_title, source_uri, snippet, permissions_scope, published_at
FROM retrieval_chunks
WHERE tenant_id = $1 AND chunk_id = ANY($2)`

	parentRows, err := s.pool.Query(ctx, fetchQuery, tenantID, parentIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch parent chunks: %w", err)
	}
	defer parentRows.Close()

	parentByChunkID := make(map[string]retrieval.EvidenceBlock)
	for parentRows.Next() {
		var b retrieval.EvidenceBlock
		if err := parentRows.Scan(
			&b.EvidenceID, &b.TenantID, &b.DocumentID, &b.DocumentVersion,
			&b.ChunkID, &b.SourceTitle, &b.SourceURI, &b.Snippet,
			&b.PermissionsScope, &b.PublishedAt,
		); err != nil {
			return nil, err
		}
		parentByChunkID[b.ChunkID] = b
	}
	if err := parentRows.Err(); err != nil {
		return nil, err
	}

	// Replace child blocks with their parents, deduplicating
	seen := make(map[string]bool)
	var result []retrieval.EvidenceBlock
	for _, block := range blocks {
		parentID, isChild := childToParent[block.ChunkID]
		if isChild {
			if seen[parentID] {
				continue
			}
			seen[parentID] = true
			if parent, ok := parentByChunkID[parentID]; ok {
				parent.Score = block.Score
				parent.RerankScore = block.RerankScore
				result = append(result, parent)
				continue
			}
			// Parent not found in DB — fall through and keep the child block
		}
		if seen[block.ChunkID] {
			continue
		}
		seen[block.ChunkID] = true
		result = append(result, block)
	}

	return result, nil
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

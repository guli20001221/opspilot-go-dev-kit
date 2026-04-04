package ingestion

import (
	"context"
	"fmt"

	"opspilot-go/internal/retrieval"
)

// Indexer persists chunks with embeddings and parent-child linking.
type Indexer struct {
	store    ChunkStore
	embedder retrieval.Embedder
}

// NewIndexer constructs the indexer.
func NewIndexer(store ChunkStore, embedder retrieval.Embedder) *Indexer {
	return &Indexer{store: store, embedder: embedder}
}

// Index embeds and persists parent and child chunks for a document.
func (idx *Indexer) Index(ctx context.Context, doc Document, chunks []Chunk) (parentCount, childCount int, err error) {
	for i, chunk := range chunks {
		if ctx.Err() != nil {
			return parentCount, childCount, ctx.Err()
		}

		parentChunkID := fmt.Sprintf("doc-%s-p%d", doc.DocumentID, i)

		// Embed and store parent chunk
		parentText := chunk.ContextPrefix + "\n\n" + chunk.Text
		parentEmb, err := idx.embedder.Embed(ctx, parentText)
		if err != nil {
			return parentCount, childCount, fmt.Errorf("embed parent chunk %d: %w", i, err)
		}

		if _, err := idx.store.UpsertWithHybrid(ctx, ChunkRecord{
			ID:               fmt.Sprintf("rc-%s-p%d", doc.DocumentID, i),
			TenantID:         doc.TenantID,
			DocumentID:       doc.DocumentID,
			DocumentVersion:  doc.DocumentVersion,
			ChunkID:          parentChunkID,
			ParentChunkID:    nil,
			SourceTitle:      doc.SourceTitle,
			SourceURI:        doc.SourceURI,
			Snippet:          chunk.Text,
			ContextPrefix:    chunk.ContextPrefix,
			Embedding:        parentEmb,
			PermissionsScope: doc.PermissionsScope,
			PublishedAt:      doc.PublishedAt,
		}); err != nil {
			return parentCount, childCount, fmt.Errorf("upsert parent chunk %d: %w", i, err)
		}
		parentCount++

		// Embed and store child chunks (one per sentence)
		for j, sentence := range chunk.Sentences {
			if ctx.Err() != nil {
				return parentCount, childCount, ctx.Err()
			}

			childSnippet := chunk.ContextPrefix + "\n\n" + sentence.Text
			childEmb, err := idx.embedder.Embed(ctx, childSnippet)
			if err != nil {
				return parentCount, childCount, fmt.Errorf("embed child chunk %d-%d: %w", i, j, err)
			}

			childChunkID := fmt.Sprintf("doc-%s-p%d-c%d", doc.DocumentID, i, j)
			parentRef := parentChunkID
			if _, err := idx.store.UpsertWithHybrid(ctx, ChunkRecord{
				ID:               fmt.Sprintf("rc-%s-p%d-c%d", doc.DocumentID, i, j),
				TenantID:         doc.TenantID,
				DocumentID:       doc.DocumentID,
				DocumentVersion:  doc.DocumentVersion,
				ChunkID:          childChunkID,
				ParentChunkID:    &parentRef,
				SourceTitle:      doc.SourceTitle,
				SourceURI:        doc.SourceURI,
				Snippet:          sentence.Text,
				ContextPrefix:    chunk.ContextPrefix,
				Embedding:        childEmb,
				PermissionsScope: doc.PermissionsScope,
				PublishedAt:      doc.PublishedAt,
			}); err != nil {
				return parentCount, childCount, fmt.Errorf("upsert child chunk %d-%d: %w", i, j, err)
			}
			childCount++
		}
	}

	return parentCount, childCount, nil
}

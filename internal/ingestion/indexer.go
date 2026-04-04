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

		// Embed parent chunk (new composite text, not cached)
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

		// Store child chunks — reuse cached sentence embeddings from chunker
		for j, sentence := range chunk.Sentences {
			if ctx.Err() != nil {
				return parentCount, childCount, ctx.Err()
			}

			childEmb := sentence.Embedding
			if len(childEmb) == 0 {
				// Fallback: embed if not cached (shouldn't happen in normal pipeline flow)
				var embErr error
				childEmb, embErr = idx.embedder.Embed(ctx, chunk.ContextPrefix+"\n\n"+sentence.Text)
				if embErr != nil {
					return parentCount, childCount, fmt.Errorf("embed child chunk %d-%d: %w", i, j, embErr)
				}
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

package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"opspilot-go/internal/llm"
	"opspilot-go/internal/retrieval"
)

// PipelineOptions configures the ingestion pipeline.
type PipelineOptions struct {
	Chunker        ChunkerOptions
	Prefixer       PrefixerOptions
	MinSentenceLen int
}

// Pipeline orchestrates the full document ingestion flow.
type Pipeline struct {
	splitter *SentenceSplitter
	chunker  *SemanticChunker
	prefixer *ContextPrefixer
	indexer  *Indexer
}

// NewPipeline constructs the ingestion pipeline with all dependencies.
func NewPipeline(embedder retrieval.Embedder, llmProvider llm.Provider, store ChunkStore, opts PipelineOptions) *Pipeline {
	return &Pipeline{
		splitter: &SentenceSplitter{MinLen: opts.MinSentenceLen},
		chunker:  NewSemanticChunker(embedder, opts.Chunker),
		prefixer: NewContextPrefixer(llmProvider, opts.Prefixer),
		indexer:  NewIndexer(store, embedder),
	}
}

// Ingest processes a single document through the full pipeline.
func (p *Pipeline) Ingest(ctx context.Context, doc Document) (IngestResult, error) {
	if doc.DocumentID == "" || doc.TenantID == "" || doc.Content == "" {
		return IngestResult{}, fmt.Errorf("document_id, tenant_id, and content are required")
	}

	start := time.Now()

	// Stage 1: Sentence splitting
	sentences := p.splitter.Split(doc.Content)
	slog.Info("ingestion: sentences split",
		slog.String("document_id", doc.DocumentID),
		slog.Int("count", len(sentences)),
		slog.Duration("duration", time.Since(start)),
	)

	if len(sentences) == 0 {
		return IngestResult{DocumentID: doc.DocumentID, TenantID: doc.TenantID}, nil
	}

	// Stage 2: Semantic chunking
	chunkStart := time.Now()
	chunks, err := p.chunker.Chunk(ctx, sentences)
	if err != nil {
		return IngestResult{}, fmt.Errorf("semantic chunking: %w", err)
	}
	slog.Info("ingestion: semantic chunking done",
		slog.String("document_id", doc.DocumentID),
		slog.Int("chunks", len(chunks)),
		slog.Duration("duration", time.Since(chunkStart)),
	)

	// Stage 3: Contextual prefix generation
	prefixStart := time.Now()
	chunks, err = p.prefixer.Prefix(ctx, doc, chunks)
	if err != nil {
		return IngestResult{}, fmt.Errorf("contextual prefix: %w", err)
	}
	slog.Info("ingestion: contextual prefixes generated",
		slog.String("document_id", doc.DocumentID),
		slog.Int("chunks", len(chunks)),
		slog.Duration("duration", time.Since(prefixStart)),
	)

	// Stage 4: Hybrid indexing with parent-child linking
	indexStart := time.Now()
	parentCount, childCount, err := p.indexer.Index(ctx, doc, chunks)
	if err != nil {
		return IngestResult{}, fmt.Errorf("indexing: %w", err)
	}
	slog.Info("ingestion: indexing complete",
		slog.String("document_id", doc.DocumentID),
		slog.Int("parents", parentCount),
		slog.Int("children", childCount),
		slog.Duration("duration", time.Since(indexStart)),
		slog.Duration("total", time.Since(start)),
	)

	return IngestResult{
		DocumentID:   doc.DocumentID,
		TenantID:     doc.TenantID,
		ChunksStored: parentCount + childCount,
		ParentChunks: parentCount,
		ChildChunks:  childCount,
	}, nil
}

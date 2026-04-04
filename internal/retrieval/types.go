package retrieval

import (
	"context"
	"time"
)

// Searcher performs structured retrieval searches.
type Searcher interface {
	Search(ctx context.Context, req RetrievalRequest) (RetrievalResult, error)
}

// RetrievalRequest is the structured query object used for retrieval.
type RetrievalRequest struct {
	RequestID      string
	TraceID        string
	TenantID       string
	SessionID      string
	PlanID         string
	QueryText      string
	RewrittenQuery string
	Filters        RetrievalFilters
	TopK           int
	UseRerank      bool
}

// RetrievalFilters contains optional retrieval filters.
type RetrievalFilters struct {
	DocumentTags []string
	TimeFrom     *time.Time
	TimeTo       *time.Time
}

// RetrievalResult is the typed retrieval output consumed upstream.
type RetrievalResult struct {
	RequestID        string
	PlanID           string
	QueryUsed        string
	EvidenceBlocks   []EvidenceBlock
	CoverageScore    float64
	MissingQuestions []string
}

// EvidenceBlock is one provenance-bearing evidence item.
type EvidenceBlock struct {
	EvidenceID       string
	TenantID         string
	DocumentID       string
	DocumentVersion  int
	ChunkID          string
	SourceTitle      string
	SourceURI        string
	Snippet          string
	Score            float64
	RerankScore      float64
	PublishedAt      *time.Time
	CitationLabel    string
	PermissionsScope string
}

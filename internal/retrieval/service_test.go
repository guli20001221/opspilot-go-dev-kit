package retrieval

import (
	"context"
	"fmt"
	"testing"
)

func TestServiceSearchCapsTopKAndPreservesTenantScope(t *testing.T) {
	catalog := make([]EvidenceBlock, 0, 15)
	for i := 0; i < 15; i++ {
		catalog = append(catalog, EvidenceBlock{
			EvidenceID:       fmt.Sprintf("evidence-%02d", i),
			TenantID:         "tenant-1",
			DocumentID:       fmt.Sprintf("doc-%02d", i),
			DocumentVersion:  1,
			ChunkID:          fmt.Sprintf("chunk-%02d", i),
			SourceTitle:      "Incident SOP",
			SourceURI:        fmt.Sprintf("kb://incident-sop/%02d", i),
			Snippet:          "incident handling procedure",
			PermissionsScope: "tenant:tenant-1",
		})
	}
	svc := NewService(catalog)

	got, err := svc.Search(context.Background(), RetrievalRequest{
		RequestID: "req-1",
		TraceID:   "trace-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		PlanID:    "plan-1",
		QueryText: "incident",
		TopK:      20,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(got.EvidenceBlocks) != 12 {
		t.Fatalf("len(EvidenceBlocks) = %d, want %d", len(got.EvidenceBlocks), 12)
	}
	for _, block := range got.EvidenceBlocks {
		if block.TenantID != "tenant-1" {
			t.Fatalf("block.TenantID = %q, want %q", block.TenantID, "tenant-1")
		}
		if block.PermissionsScope == "" {
			t.Fatal("block.PermissionsScope is empty")
		}
		if block.CitationLabel == "" {
			t.Fatal("block.CitationLabel is empty")
		}
	}
}

func TestServiceSearchUsesRewrittenQueryAndReturnsProvenance(t *testing.T) {
	svc := NewService([]EvidenceBlock{
		{
			EvidenceID:       "evidence-1",
			TenantID:         "tenant-1",
			DocumentID:       "doc-1",
			DocumentVersion:  2,
			ChunkID:          "chunk-1",
			SourceTitle:      "Ticket Summary",
			SourceURI:        "kb://tickets/1",
			Snippet:          "ticket timeline and incident summary",
			PermissionsScope: "tenant:tenant-1",
		},
	})

	got, err := svc.Search(context.Background(), RetrievalRequest{
		RequestID:      "req-2",
		TraceID:        "trace-2",
		TenantID:       "tenant-1",
		SessionID:      "session-1",
		PlanID:         "plan-2",
		QueryText:      "noise",
		RewrittenQuery: "ticket summary",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got.QueryUsed != "ticket summary" {
		t.Fatalf("QueryUsed = %q, want %q", got.QueryUsed, "ticket summary")
	}
	if len(got.EvidenceBlocks) != 1 {
		t.Fatalf("len(EvidenceBlocks) = %d, want %d", len(got.EvidenceBlocks), 1)
	}
	block := got.EvidenceBlocks[0]
	if block.EvidenceID == "" || block.DocumentID == "" || block.ChunkID == "" {
		t.Fatalf("missing provenance fields in %#v", block)
	}
	if block.CitationLabel != "[1]" {
		t.Fatalf("CitationLabel = %q, want %q", block.CitationLabel, "[1]")
	}
}

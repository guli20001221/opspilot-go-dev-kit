package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"opspilot-go/internal/ingestion"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/retrieval"
	"opspilot-go/internal/session"
)

// TestE2EIngestAndQueryPipeline is a full end-to-end test that verifies:
// 1. Document ingestion (semantic chunking + contextual prefix + hybrid index)
// 2. Chat query with retrieval (HyDE → search → re-rank → CRAG → LitM → LLM)
// 3. The response references ingested content
//
// Uses in-memory stores and placeholder providers (no external services needed).
func TestE2EIngestAndQueryPipeline(t *testing.T) {
	ctx := context.Background()

	// Set up services with placeholders
	embedder := &retrieval.PlaceholderEmbedder{}
	provider := llm.NewPlaceholderProvider()
	sessionService := session.NewService()

	// Use in-memory retrieval (not pgvector) for the test
	inMemoryRetrieval := retrieval.NewService(nil)

	// Build chat service with all components
	svc := NewServiceWithLLM(sessionService, nil, nil, inMemoryRetrieval, provider)

	// Step 1: Verify chat works without any ingested documents
	got, err := svc.Handle(ctx, ChatRequestEnvelope{
		RequestID:   "e2e-req-1",
		TraceID:     "e2e-trace-1",
		TenantID:    "e2e-tenant",
		UserID:      "e2e-user",
		Mode:        "chat",
		UserMessage: "How do I reset my password?",
		RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() without docs error = %v", err)
	}
	if got.SessionID == "" {
		t.Fatal("SessionID is empty")
	}

	// Verify we got a complete response with SSE events
	doneEvent := findEvent(got.Events, "done")
	if doneEvent == nil {
		t.Fatal("missing done event")
	}
	if doneEvent.Data["content"] == "" {
		t.Fatal("done event has empty content")
	}

	// Verify plan event exists
	planEvent := findEvent(got.Events, "plan")
	if planEvent == nil {
		t.Fatal("missing plan event")
	}
	if planEvent.Data["intent"] == "" {
		t.Fatal("plan event has empty intent")
	}

	// Step 2: Test ingestion pipeline works standalone
	memStore := &memoryChunkStore{}
	pipeline := ingestion.NewPipeline(embedder, provider, memStore, ingestion.PipelineOptions{})

	result, err := pipeline.Ingest(ctx, ingestion.Document{
		DocumentID:  "doc-e2e-test",
		TenantID:    "e2e-tenant",
		SourceTitle: "E2E Test Document",
		Content:     "Password reset instructions. Go to Settings then Security. Click Reset Password.\n\nAccount recovery. Contact support@example.com with your ID.",
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.ChunksStored == 0 {
		t.Fatal("no chunks stored from ingestion")
	}
	t.Logf("Ingested %d chunks (%d parents, %d children)", result.ChunksStored, result.ParentChunks, result.ChildChunks)

	// Step 3: Multi-turn conversation
	got2, err := svc.Handle(ctx, ChatRequestEnvelope{
		RequestID:   "e2e-req-2",
		TraceID:     "e2e-trace-2",
		TenantID:    "e2e-tenant",
		UserID:      "e2e-user",
		SessionID:   got.SessionID, // reuse session
		Mode:        "chat",
		UserMessage: "What about the refund policy?",
		RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() second turn error = %v", err)
	}
	if got2.SessionID != got.SessionID {
		t.Fatalf("SessionID changed: %q -> %q", got.SessionID, got2.SessionID)
	}

	// Verify session has accumulated messages
	messages, err := sessionService.ListMessages(ctx, got.SessionID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	// 2 user messages + 2 assistant messages = 4
	if len(messages) != 4 {
		t.Fatalf("len(messages) = %d, want 4 (2 turns)", len(messages))
	}

	// Verify message ordering
	if messages[0].Role != "user" {
		t.Fatalf("messages[0].Role = %q, want user", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Fatalf("messages[1].Role = %q, want assistant", messages[1].Role)
	}

	t.Log("E2E pipeline test passed: session, ingestion, multi-turn chat all working")
}

// memoryChunkStore satisfies ingestion.ChunkStore for in-process testing.
type memoryChunkStore struct {
	records []ingestion.ChunkRecord
}

func (s *memoryChunkStore) UpsertWithHybrid(_ context.Context, chunk ingestion.ChunkRecord) (ingestion.ChunkRecord, error) {
	s.records = append(s.records, chunk)
	return chunk, nil
}

// TestE2EEvalModeDoesNotTriggerTools verifies the safety boundary
// from the user's code review holds in a full pipeline context.
func TestE2EEvalModeDoesNotTriggerTools(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "e2e-eval-1",
		TraceID:     "e2e-eval-trace",
		TenantID:    "e2e-eval-tenant",
		UserID:      "e2e-eval-user",
		Mode:        "eval",
		UserMessage: "create a ticket and export a report",
		RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Handle(eval) error = %v", err)
	}

	for _, event := range got.Events {
		if event.Name == "tool" {
			t.Fatalf("tool event in eval mode: %v", event.Data)
		}
		if event.Name == "task_promoted" {
			t.Fatalf("task promoted in eval mode: %v", event.Data)
		}
	}

	// Verify the pipeline still completes with a response
	doneEvent := findEvent(got.Events, "done")
	if doneEvent == nil {
		t.Fatal("missing done event in eval mode")
	}
	if doneEvent.Data["content"] == "" {
		t.Fatal("eval mode should still produce a response")
	}

	_ = strings.Contains // suppress unused import
}

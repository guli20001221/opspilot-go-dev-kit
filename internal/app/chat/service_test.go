package chat

import (
	"context"
	"testing"
	"time"

	"opspilot-go/internal/session"
)

func TestServiceHandleCreatesSessionAndBuildsStreamEvents(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-1",
		TraceID:     "trace-1",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "hello",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got.SessionID == "" {
		t.Fatal("Handle() returned empty session ID")
	}
	if len(got.Context.Planner.Blocks) == 0 {
		t.Fatal("Handle() returned empty planner context blocks")
	}
	if got.Context.Log.RequestID != "req-1" {
		t.Fatalf("Context.Log.RequestID = %q, want %q", got.Context.Log.RequestID, "req-1")
	}
	if got.Plan.Intent != "knowledge_qa" {
		t.Fatalf("Plan.Intent = %q, want %q", got.Plan.Intent, "knowledge_qa")
	}
	if len(got.Plan.Steps) == 0 {
		t.Fatal("Handle() returned empty plan steps")
	}
	if got.Retrieval.QueryUsed != "hello" {
		t.Fatalf("Retrieval.QueryUsed = %q, want %q", got.Retrieval.QueryUsed, "hello")
	}
	if len(got.Events) != 3 {
		t.Fatalf("len(Events) = %d, want %d", len(got.Events), 3)
	}

	assertEventPayload(t, got.Events[0], "meta", map[string]string{
		"request_id": "req-1",
		"trace_id":   "trace-1",
		"session_id": got.SessionID,
	})
	assertEventPayload(t, got.Events[1], "state", map[string]string{
		"state": "completed",
	})
	assertEventPayload(t, got.Events[2], "done", map[string]string{
		"session_id": got.SessionID,
		"content":    PlaceholderAssistantResponse,
	})

	messages, err := sessionService.ListMessages(context.Background(), got.SessionID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 2)
	}
	if messages[0].Role != session.RoleUser || messages[0].Content != "hello" {
		t.Fatalf("messages[0] = %#v, want user hello", messages[0])
	}
	if messages[1].Role != session.RoleAssistant || messages[1].Content != PlaceholderAssistantResponse {
		t.Fatalf("messages[1] = %#v, want assistant placeholder", messages[1])
	}
}

func TestServiceHandleExecutesReadOnlyToolForTicketQuery(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-tool",
		TraceID:     "trace-tool",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "search related ticket history",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if !got.Plan.RequiresTool {
		t.Fatal("Plan.RequiresTool = false, want true")
	}
	if len(got.ToolResults) != 1 {
		t.Fatalf("len(ToolResults) = %d, want %d", len(got.ToolResults), 1)
	}
	if got.ToolResults[0].ToolName != "ticket_search" {
		t.Fatalf("ToolResults[0].ToolName = %q, want %q", got.ToolResults[0].ToolName, "ticket_search")
	}
	if got.ToolResults[0].Status != "succeeded" {
		t.Fatalf("ToolResults[0].Status = %q, want %q", got.ToolResults[0].Status, "succeeded")
	}
}

func assertEventPayload(t *testing.T, got StreamEvent, wantName string, wantPayload map[string]string) {
	t.Helper()

	if got.Name != wantName {
		t.Fatalf("event.Name = %q, want %q", got.Name, wantName)
	}
	if len(got.Data) != len(wantPayload) {
		t.Fatalf("len(event.Data) = %d, want %d", len(got.Data), len(wantPayload))
	}
	for key, wantValue := range wantPayload {
		if got.Data[key] != wantValue {
			t.Fatalf("event.Data[%q] = %q, want %q", key, got.Data[key], wantValue)
		}
	}
}

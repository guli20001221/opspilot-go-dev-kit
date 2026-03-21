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
	if got.Critic.Verdict != "revise" {
		t.Fatalf("Critic.Verdict = %q, want %q", got.Critic.Verdict, "revise")
	}
	if len(got.Events) != 5 {
		t.Fatalf("len(Events) = %d, want %d", len(got.Events), 5)
	}

	assertEventPayload(t, got.Events[0], "meta", map[string]string{
		"request_id": "req-1",
		"trace_id":   "trace-1",
		"session_id": got.SessionID,
	})
	assertEventPayload(t, got.Events[1], "plan", map[string]string{
		"intent": "knowledge_qa",
	})
	assertEventPayload(t, got.Events[2], "retrieval", map[string]string{
		"query_used": "hello",
	})
	assertEventPayload(t, got.Events[3], "state", map[string]string{
		"state": "completed",
	})
	assertEventPayload(t, got.Events[4], "done", map[string]string{
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
	if got.Critic.Verdict != "approve" {
		t.Fatalf("Critic.Verdict = %q, want %q", got.Critic.Verdict, "approve")
	}
	if got.Events[2].Name != "tool" {
		t.Fatalf("Events[2].Name = %q, want %q", got.Events[2].Name, "tool")
	}
}

func TestServiceHandlePromotesTaskModeIntoWorkflowResult(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-task",
		TraceID:     "trace-task",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "task",
		UserMessage: "generate a report for last week's incidents",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if !got.Plan.RequiresWorkflow {
		t.Fatal("Plan.RequiresWorkflow = false, want true")
	}
	if got.PromotedTask == nil {
		t.Fatal("PromotedTask is nil")
	}
	if got.PromotedTask.Status != "queued" {
		t.Fatalf("PromotedTask.Status = %q, want %q", got.PromotedTask.Status, "queued")
	}
	foundPromotionEvent := false
	for _, event := range got.Events {
		if event.Name == "task_promoted" {
			foundPromotionEvent = true
			break
		}
	}
	if !foundPromotionEvent {
		t.Fatal("task_promoted event not found")
	}
}

func assertEventPayload(t *testing.T, got StreamEvent, wantName string, wantPayload map[string]string) {
	t.Helper()

	if got.Name != wantName {
		t.Fatalf("event.Name = %q, want %q", got.Name, wantName)
	}
	for key, wantValue := range wantPayload {
		if got.Data[key] != wantValue {
			t.Fatalf("event.Data[%q] = %q, want %q", key, got.Data[key], wantValue)
		}
	}
}

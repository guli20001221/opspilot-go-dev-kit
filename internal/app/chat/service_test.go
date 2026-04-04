package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"opspilot-go/internal/llm"
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

func TestServiceHandlePromotesApprovedToolTaskWithPayload(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-approved-tool",
		TraceID:     "trace-approved-tool",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "comment on ticket INC-100 with approved note",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got.PromotedTask == nil {
		t.Fatal("PromotedTask is nil")
	}
	if got.PromotedTask.TaskType != "approved_tool_execution" {
		t.Fatalf("PromotedTask.TaskType = %q, want %q", got.PromotedTask.TaskType, "approved_tool_execution")
	}
	if got.PromotedTask.ToolName != "ticket_comment_create" {
		t.Fatalf("PromotedTask.ToolName = %q, want %q", got.PromotedTask.ToolName, "ticket_comment_create")
	}

	var args map[string]string
	if err := json.Unmarshal(got.PromotedTask.ToolArguments, &args); err != nil {
		t.Fatalf("Unmarshal(ToolArguments) error = %v", err)
	}
	if args["ticket_id"] != "INC-100" {
		t.Fatalf("ToolArguments.ticket_id = %q, want %q", args["ticket_id"], "INC-100")
	}
	if args["comment"] != "comment on ticket INC-100 with approved note" {
		t.Fatalf("ToolArguments.comment = %q, want original user message", args["comment"])
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

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	content string
	err     error
}

func (m *mockProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	if m.err != nil {
		return llm.CompletionResponse{}, m.err
	}
	return llm.CompletionResponse{Content: m.content, Model: "mock"}, nil
}

func TestServiceHandleWithLLMProviderUsesProviderResponse(t *testing.T) {
	sessionService := session.NewService()
	provider := &mockProvider{content: "Hello from the LLM!"}
	svc := NewServiceWithLLM(sessionService, nil, nil, nil, provider)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-llm-1",
		TraceID:     "trace-llm-1",
		TenantID:    "tenant-llm",
		UserID:      "user-llm",
		Mode:        "chat",
		UserMessage: "What is OpsPilot?",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Check done event carries LLM response
	doneEvent := findEvent(got.Events, "done")
	if doneEvent == nil {
		t.Fatal("missing done event")
	}
	if doneEvent.Data["content"] != "Hello from the LLM!" {
		t.Fatalf("done.content = %q, want %q", doneEvent.Data["content"], "Hello from the LLM!")
	}

	// Check assistant message stored in session
	messages, err := sessionService.ListMessages(context.Background(), got.SessionID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	var assistantContent string
	for _, msg := range messages {
		if msg.Role == session.RoleAssistant {
			assistantContent = msg.Content
		}
	}
	if assistantContent != "Hello from the LLM!" {
		t.Fatalf("stored assistant content = %q, want %q", assistantContent, "Hello from the LLM!")
	}
}

func TestServiceHandleFallsBackWhenLLMErrors(t *testing.T) {
	sessionService := session.NewService()
	provider := &mockProvider{err: fmt.Errorf("provider unavailable")}
	svc := NewServiceWithLLM(sessionService, nil, nil, nil, provider)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-llm-err",
		TraceID:     "trace-llm-err",
		TenantID:    "tenant-llm-err",
		UserID:      "user-llm-err",
		Mode:        "chat",
		UserMessage: "Test fallback",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	doneEvent := findEvent(got.Events, "done")
	if doneEvent == nil {
		t.Fatal("missing done event")
	}
	if doneEvent.Data["content"] != PlaceholderAssistantResponse {
		t.Fatalf("done.content = %q, want placeholder %q", doneEvent.Data["content"], PlaceholderAssistantResponse)
	}
}

func findEvent(events []StreamEvent, name string) *StreamEvent {
	for i := range events {
		if events[i].Name == name {
			return &events[i]
		}
	}
	return nil
}

func TestServiceHandleEvalModeSkipsToolExecutionAndWorkflowPromotion(t *testing.T) {
	sessionService := session.NewService()
	// Use default service (with tool registry) — keyword planner will create
	// tool steps for ticket-related messages
	svc := NewService(sessionService)

	// "create a ticket" triggers incident_assist intent with tool steps
	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-eval-safety",
		TraceID:     "trace-eval-safety",
		TenantID:    "tenant-eval-safety",
		UserID:      "user-eval-safety",
		Mode:        "eval",
		UserMessage: "create a ticket for the outage",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Verify no tool events were emitted (tools skipped in eval mode)
	for _, event := range got.Events {
		if event.Name == "tool" {
			t.Fatalf("tool event emitted in eval mode: %v — tools should be skipped", event.Data)
		}
	}

	// Verify no task was promoted
	if got.PromotedTask != nil {
		t.Fatalf("PromotedTask = %v, want nil in eval mode", got.PromotedTask)
	}

	// Verify session was still created and assistant message recorded
	if got.SessionID == "" {
		t.Fatal("SessionID is empty — session should still be created in eval mode")
	}

	doneEvent := findEvent(got.Events, "done")
	if doneEvent == nil {
		t.Fatal("missing done event")
	}
	if doneEvent.Data["content"] == "" {
		t.Fatal("done event content is empty — LLM response should still be generated")
	}
}

func TestServiceHandleNormalModeAllowsToolExecution(t *testing.T) {
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-normal-tools",
		TraceID:     "trace-normal-tools",
		TenantID:    "tenant-normal-tools",
		UserID:      "user-normal-tools",
		Mode:        "chat",
		UserMessage: "create a ticket for the outage",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// In normal mode, tool events should be emitted for ticket-related messages
	hasToolEvent := false
	for _, event := range got.Events {
		if event.Name == "tool" {
			hasToolEvent = true
			break
		}
	}
	if !hasToolEvent {
		t.Fatal("no tool event emitted in normal mode — tools should execute")
	}
}

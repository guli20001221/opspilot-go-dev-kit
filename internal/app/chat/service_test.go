package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/session"
	toolregistry "opspilot-go/internal/tools/registry"
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

func TestServiceHandleWriteToolWithStructuredArgsSucceeds(t *testing.T) {
	// LLM planner produces a write tool step WITH ToolArguments — should execute.
	planJSON := `{"intent":"incident_assist","reasoning":"create comment","requires_retrieval":false,"requires_tool":true,"requires_workflow":false,"requires_approval":false,"output_schema":"markdown","steps":[{"kind":"tool","name":"create comment","depends_on":[],"tool_name":"ticket_comment_create","tool_arguments":{"ticket_id":"INC-100","comment":"automated note"},"read_only":false,"needs_approval":true},{"kind":"synthesize","name":"compose","depends_on":["create comment"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false},{"kind":"critic","name":"validate","depends_on":["compose"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false}]}`

	llmCallIdx := 0
	provider := &sequenceMockProvider{
		responses: []llm.CompletionResponse{
			{Content: planJSON, Model: "mock"},
			{Content: "Done.", Model: "mock"},
			{Content: `{"groundedness":0.9,"citation_coverage":0.8,"tool_consistency":1.0,"risk_level":"low","verdict":"approve","reasoning":"ok"}`, Model: "mock"},
		},
		callIdx: &llmCallIdx,
	}
	sessionService := session.NewService()
	svc := NewServiceWithLLM(sessionService, nil, nil, nil, provider)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-write-tool-happy",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "add a comment to ticket INC-100",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v, want nil (write tool with structured args should succeed)", err)
	}
	// The write tool has RequiresApproval=true, so it should be approval-gated
	hasToolEvent := false
	for _, evt := range got.Events {
		if evt.Name == "tool" {
			hasToolEvent = true
		}
	}
	if !hasToolEvent {
		t.Fatal("no tool event — write tool with structured args should execute (approval-gated)")
	}
}

func TestServiceHandleRejectsWriteToolWithoutStructuredArgs(t *testing.T) {
	// Keyword planner produces a write tool step without ToolArguments.
	// The safety boundary should reject execution rather than falling back
	// to heuristic argument construction from raw user text.
	sessionService := session.NewService()
	svc := NewService(sessionService)

	_, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-write-tool-safety",
		TraceID:     "trace-write-tool-safety",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "comment on ticket INC-100 with approved note",
	})
	if err == nil {
		t.Fatal("Handle() error = nil, want error for write tool without structured arguments")
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

func TestServiceHandleNormalModeAllowsReadOnlyToolExecution(t *testing.T) {
	// Read-only tools (ticket_search) still execute via heuristic fallback.
	sessionService := session.NewService()
	svc := NewService(sessionService)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-normal-tools",
		TraceID:     "trace-normal-tools",
		TenantID:    "tenant-normal-tools",
		UserID:      "user-normal-tools",
		Mode:        "chat",
		UserMessage: "search related ticket history",
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	hasToolEvent := false
	for _, event := range got.Events {
		if event.Name == "tool" {
			hasToolEvent = true
			break
		}
	}
	if !hasToolEvent {
		t.Fatal("no tool event emitted in normal mode — read-only tools should execute")
	}
}

func TestBuildToolArgumentsFallbackForKeywordPlanner(t *testing.T) {
	// Keyword planner does not produce ToolArguments — verify the fallback
	// heuristic still works correctly.
	args, err := buildToolArguments("ticket_search", "search for INC-200")
	if err != nil {
		t.Fatalf("buildToolArguments() error = %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(args, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if parsed["query"] != "search for INC-200" {
		t.Fatalf("query = %q, want user message", parsed["query"])
	}
}

func TestBuildToolArgumentsFallbackForCommentCreate(t *testing.T) {
	args, err := buildToolArguments("ticket_comment_create", "comment on ticket INC-300 fix applied")
	if err != nil {
		t.Fatalf("buildToolArguments() error = %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(args, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if parsed["ticket_id"] != "INC-300" {
		t.Fatalf("ticket_id = %q, want %q", parsed["ticket_id"], "INC-300")
	}
	if parsed["comment"] == "" {
		t.Fatal("comment is empty")
	}
}

// --- Replanning integration tests ---

func TestServiceHandleKeywordPlanDoesNotReplanOnToolError(t *testing.T) {
	// Keyword planner cannot replan — tool errors propagate directly.
	sessionService := session.NewService()
	// Use a registry with a tool that will fail
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:        "ticket_search",
		ActionClass: "read",
		ReadOnly:    true,
		Executor: func(_ context.Context, _ json.RawMessage) (any, error) {
			return nil, fmt.Errorf("simulated search failure")
		},
	})
	svc := NewServiceWithRegistry(sessionService, nil, registry)

	_, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-kw-no-replan",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "search related ticket history",
	})
	// Should get the tool error directly since keyword plans can't replan
	if err == nil {
		t.Fatal("Handle() error = nil, want tool execution error")
	}
}

func TestServiceHandleReplanPreservesFailedToolInResults(t *testing.T) {
	// Scenario: tool fails → replan succeeds → failed tool attempt must appear
	// in HandleResult.ToolResults alongside the revised plan's successful results.
	sessionService := session.NewService()

	callCount := 0
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:        "ticket_search",
		ActionClass: "read",
		ReadOnly:    true,
		Executor: func(_ context.Context, _ json.RawMessage) (any, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("simulated first-call failure")
			}
			// Second call (from revised plan) succeeds
			return map[string]string{"ticket_id": "INC-100", "summary": "found"}, nil
		},
	})

	// The mock LLM needs to return:
	// 1st call: initial plan (tool step that will fail)
	// 2nd call: replan response (fallback plan with retrieve + synthesize + critic, no tool)
	// 3rd call: LLM completion for the answer
	// But the planner uses the LLM, and the chat completion also uses it.
	// The keyword planner doesn't use LLM, but keyword plans can't replan.
	// So we need an LLM provider for the planner to produce an LLM-sourced plan.

	planJSON := `{"intent":"incident_assist","reasoning":"search tickets","requires_retrieval":false,"requires_tool":true,"requires_workflow":false,"requires_approval":false,"output_schema":"markdown","steps":[{"kind":"tool","name":"search tickets","depends_on":[],"tool_name":"ticket_search","tool_arguments":{"query":"test"},"read_only":true,"needs_approval":false},{"kind":"synthesize","name":"compose","depends_on":["search tickets"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false},{"kind":"critic","name":"validate","depends_on":["compose"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false}]}`
	replanJSON := `{"intent":"knowledge_qa","reasoning":"fallback after tool failure","requires_retrieval":true,"requires_tool":false,"requires_workflow":false,"requires_approval":false,"output_schema":"markdown","steps":[{"kind":"retrieve","name":"retrieve docs","depends_on":[],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false},{"kind":"synthesize","name":"compose answer","depends_on":["retrieve docs"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false},{"kind":"critic","name":"validate","depends_on":["compose answer"],"tool_name":"","tool_arguments":{},"read_only":false,"needs_approval":false}]}`

	llmCallIdx := 0
	provider := &sequenceMockProvider{
		responses: []llm.CompletionResponse{
			{Content: planJSON, Model: "mock"},        // initial plan
			{Content: replanJSON, Model: "mock"},       // replan
			{Content: "LLM answer after replan", Model: "mock"}, // completion
			{Content: `{"groundedness":0.8,"citation_coverage":0.7,"tool_consistency":1.0,"risk_level":"low","verdict":"approve","reasoning":"ok"}`, Model: "mock"}, // critic
		},
		callIdx: &llmCallIdx,
	}

	svc := NewServiceWithLLM(sessionService, nil, registry, nil, provider)

	got, err := svc.Handle(context.Background(), ChatRequestEnvelope{
		RequestID:   "req-replan-audit",
		TenantID:    "tenant-replan",
		UserID:      "user-1",
		Mode:        "chat",
		UserMessage: "find tickets about outage",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Verify replan occurred
	if got.ReplanCount != 1 {
		t.Fatalf("ReplanCount = %d, want 1", got.ReplanCount)
	}

	// Key assertion: the failed tool attempt must be in ToolResults
	foundFailed := false
	for _, tr := range got.ToolResults {
		if tr.Status == "failed" && tr.ToolName == "ticket_search" {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Fatal("failed tool attempt not found in ToolResults — audit trail is incomplete")
	}

	// Verify SSE events include a tool event with failed status
	foundFailedEvent := false
	for _, evt := range got.Events {
		if evt.Name == "tool" && evt.Data["status"] == "failed" {
			foundFailedEvent = true
			break
		}
	}
	if !foundFailedEvent {
		t.Fatal("failed tool SSE event not found — operator visibility is incomplete")
	}
}

// sequenceMockProvider returns different LLM responses for successive calls.
type sequenceMockProvider struct {
	responses []llm.CompletionResponse
	callIdx   *int
}

func (m *sequenceMockProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	idx := *m.callIdx
	*m.callIdx++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return llm.CompletionResponse{Content: "{}", Model: "mock-fallback"}, nil
}

func TestBuildExecutedSteps(t *testing.T) {
	results := []agenttool.ToolResult{
		{ToolCallID: "tc-1", ToolName: "ticket_search", Status: "succeeded", OutputSummary: "found 3 matches"},
		{ToolCallID: "tc-2", ToolName: "ticket_comment_create", Status: "approval_required", OutputSummary: "needs approval"},
	}
	steps := buildExecutedSteps(results)
	if len(steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(steps))
	}
	if steps[0].ToolName != "ticket_search" || steps[0].Status != "succeeded" {
		t.Fatalf("step[0] = %+v, want ticket_search/succeeded", steps[0])
	}
	if steps[1].ToolName != "ticket_comment_create" || steps[1].Status != "approval_required" {
		t.Fatalf("step[1] = %+v, want ticket_comment_create/approval_required", steps[1])
	}
}

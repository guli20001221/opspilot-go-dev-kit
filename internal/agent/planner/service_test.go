package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"opspilot-go/internal/contextengine"
	"opspilot-go/internal/llm"
)

// --- Keyword fallback tests (existing behavior preserved) ---

func TestServicePlanKnowledgeQuestion(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-1",
		TraceID:     "trace-1",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the incident SOP?",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{{Kind: contextengine.BlockKindRecentTurns, Content: "user: what is the incident SOP?"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentKnowledgeQA {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentKnowledgeQA)
	}
	if !got.RequiresRetrieval {
		t.Fatal("RequiresRetrieval = false, want true")
	}
	if got.RequiresTool {
		t.Fatal("RequiresTool = true, want false")
	}
	if got.RequiresWorkflow {
		t.Fatal("RequiresWorkflow = true, want false")
	}
	if got.MaxSteps > 6 {
		t.Fatalf("MaxSteps = %d, want <= 6", got.MaxSteps)
	}
	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q", got.Source, PlanSourceKeyword)
	}
	assertStepKinds(t, got.Steps, StepKindRetrieve, StepKindSynthesize, StepKindCritic)
}

func TestServicePlanTaskRequestPromotesWorkflow(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-2",
		TraceID:     "trace-2",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "task",
		UserMessage: "generate a report for last week's incidents",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{{Kind: contextengine.BlockKindUserProfile, Content: "tenant_id=tenant-1"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentReportRequest {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentReportRequest)
	}
	if !got.RequiresWorkflow {
		t.Fatal("RequiresWorkflow = false, want true")
	}
	if len(got.Steps) == 0 || got.Steps[0].Kind != StepKindPromoteWorkflow {
		t.Fatalf("first step = %#v, want promote_workflow", got.Steps)
	}
}

func TestServicePlanTicketSearchUsesReadOnlyTool(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-3",
		TraceID:     "trace-3",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "search related ticket history",
		Context:     contextengine.PlannerContext{},
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_search", ReadOnly: true},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !got.RequiresTool {
		t.Fatal("RequiresTool = false, want true")
	}
	if got.RequiresApproval {
		t.Fatal("RequiresApproval = true, want false")
	}
	assertToolStep(t, got.Steps, "ticket_search", true)
}

// --- LLM-backed planner tests ---

// mockLLMProvider implements llm.Provider for testing structured plan output.
type mockLLMProvider struct {
	response llm.CompletionResponse
	err      error
	captured *llm.CompletionRequest
}

func (m *mockLLMProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	m.captured = &req
	if m.err != nil {
		return llm.CompletionResponse{}, m.err
	}
	return m.response, nil
}

func buildMockPlanJSON(intent string, steps []llmPlanStep) string {
	resp := llmPlanResponse{
		Intent:            intent,
		Reasoning:         "test reasoning for " + intent,
		RequiresRetrieval: false,
		RequiresTool:      false,
		RequiresWorkflow:  false,
		RequiresApproval:  false,
		OutputSchema:      "markdown",
		Steps:             steps,
	}

	for _, s := range steps {
		if s.Kind == StepKindRetrieve {
			resp.RequiresRetrieval = true
		}
		if s.Kind == StepKindTool {
			resp.RequiresTool = true
			if s.NeedsApproval {
				resp.RequiresApproval = true
			}
		}
		if s.Kind == StepKindPromoteWorkflow {
			resp.RequiresWorkflow = true
		}
	}
	if intent == IntentReportRequest {
		resp.OutputSchema = "structured_summary"
		resp.RequiresWorkflow = true
	}

	data, _ := json.Marshal(resp)
	return string(data)
}

func TestServicePlanWithLLMKnowledgeQA(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentKnowledgeQA, []llmPlanStep{
		{Kind: StepKindRetrieve, Name: "retrieve docs"},
		{Kind: StepKindSynthesize, Name: "compose answer", DependsOn: []string{"retrieve docs"}},
		{Kind: StepKindCritic, Name: "validate", DependsOn: []string{"compose answer"}},
	})

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-llm-1",
		TraceID:     "trace-llm-1",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the deployment process?",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{{Kind: contextengine.BlockKindRecentTurns, Content: "user: what is the deployment process?"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentKnowledgeQA {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentKnowledgeQA)
	}
	if got.Source != PlanSourceLLM {
		t.Fatalf("Source = %q, want %q", got.Source, PlanSourceLLM)
	}
	if got.PromptVersion != PromptVersion {
		t.Fatalf("PromptVersion = %q, want %q", got.PromptVersion, PromptVersion)
	}
	if !got.RequiresRetrieval {
		t.Fatal("RequiresRetrieval = false, want true")
	}
	if got.PlannerReasoningShort == "" {
		t.Fatal("PlannerReasoningShort is empty")
	}
	assertStepKinds(t, got.Steps, StepKindRetrieve, StepKindSynthesize, StepKindCritic)

	// Verify the LLM request was properly constructed.
	if provider.captured == nil {
		t.Fatal("LLM provider was not called")
	}
	if provider.captured.ResponseFormat != llm.ResponseFormatJSON {
		t.Fatalf("ResponseFormat = %q, want %q", provider.captured.ResponseFormat, llm.ResponseFormatJSON)
	}
	if provider.captured.SystemPrompt == "" {
		t.Fatal("SystemPrompt is empty")
	}
}

func TestServicePlanWithLLMToolSelection(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentIncidentAssist, []llmPlanStep{
		{Kind: StepKindTool, Name: "search tickets", ToolName: "ticket_search", ReadOnly: true},
		{Kind: StepKindSynthesize, Name: "compose answer", DependsOn: []string{"search tickets"}},
		{Kind: StepKindCritic, Name: "validate", DependsOn: []string{"compose answer"}},
	})

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-llm-tool",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "find tickets related to the database outage",
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_search", ReadOnly: true},
			{Name: "ticket_comment_create", ReadOnly: false, RequiresApproval: true},
		},
		TenantPolicy: TenantPolicy{AllowToolUse: true},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentIncidentAssist {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentIncidentAssist)
	}
	if !got.RequiresTool {
		t.Fatal("RequiresTool = false, want true")
	}
	if got.RequiresApproval {
		t.Fatal("RequiresApproval = true, want false for read-only tool")
	}
	if got.Source != PlanSourceLLM {
		t.Fatalf("Source = %q, want %q", got.Source, PlanSourceLLM)
	}
	assertToolStep(t, got.Steps, "ticket_search", true)
}

func TestServicePlanWithLLMWorkflowPromotion(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentReportRequest, []llmPlanStep{
		{Kind: StepKindPromoteWorkflow, Name: "promote to workflow"},
	})

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-llm-wf",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "task",
		UserMessage: "generate an incident summary report",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentReportRequest {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentReportRequest)
	}
	if !got.RequiresWorkflow {
		t.Fatal("RequiresWorkflow = false, want true")
	}
	if got.OutputSchema != "structured_summary" {
		t.Fatalf("OutputSchema = %q, want %q", got.OutputSchema, "structured_summary")
	}
	if len(got.Steps) != 1 || got.Steps[0].Kind != StepKindPromoteWorkflow {
		t.Fatalf("Steps = %#v, want single promote_workflow step", got.Steps)
	}
}

func TestServicePlanFallsBackOnLLMError(t *testing.T) {
	provider := &mockLLMProvider{
		err: fmt.Errorf("provider unavailable"),
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-fallback",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the incident SOP?",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// Should have fallen back to keyword planner
	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q (fallback)", got.Source, PlanSourceKeyword)
	}
	if got.Intent != IntentKnowledgeQA {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentKnowledgeQA)
	}
	if !got.RequiresRetrieval {
		t.Fatal("RequiresRetrieval = false, want true")
	}
}

func TestServicePlanFallsBackOnInvalidJSON(t *testing.T) {
	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: "not valid json at all", Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-bad-json",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the incident SOP?",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q (fallback)", got.Source, PlanSourceKeyword)
	}
}

func TestServicePlanFallsBackOnInvalidIntent(t *testing.T) {
	badPlan := `{"intent":"invalid_intent","reasoning":"test","requires_retrieval":false,"requires_tool":false,"requires_workflow":false,"requires_approval":false,"output_schema":"markdown","steps":[{"kind":"retrieve","name":"test","depends_on":[],"tool_name":"","read_only":false,"needs_approval":false}]}`
	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: badPlan, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-bad-intent",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q (fallback)", got.Source, PlanSourceKeyword)
	}
}

func TestServicePlanFallsBackOnEmptySteps(t *testing.T) {
	emptyPlan := `{"intent":"knowledge_qa","reasoning":"test","requires_retrieval":true,"requires_tool":false,"requires_workflow":false,"requires_approval":false,"output_schema":"markdown","steps":[]}`
	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: emptyPlan, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-empty-steps",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q (fallback)", got.Source, PlanSourceKeyword)
	}
}

func TestServicePlanWithLLMStripsCodeFences(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentKnowledgeQA, []llmPlanStep{
		{Kind: StepKindRetrieve, Name: "retrieve"},
		{Kind: StepKindSynthesize, Name: "compose"},
		{Kind: StepKindCritic, Name: "validate"},
	})
	wrapped := "```json\n" + planJSON + "\n```"

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: wrapped, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-fences",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the deployment process?",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceLLM {
		t.Fatalf("Source = %q, want %q", got.Source, PlanSourceLLM)
	}
	if got.Intent != IntentKnowledgeQA {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentKnowledgeQA)
	}
}

func TestServicePlanWithPlaceholderProviderUsesKeywordFallback(t *testing.T) {
	svc := NewServiceWithLLM(llm.NewPlaceholderProvider())

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-placeholder",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the SOP?",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q (placeholder should use keyword fallback)", got.Source, PlanSourceKeyword)
	}
}

func TestServicePlanWithNilProviderUsesKeywordFallback(t *testing.T) {
	svc := NewServiceWithLLM(nil)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-nil",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Source != PlanSourceKeyword {
		t.Fatalf("Source = %q, want %q", got.Source, PlanSourceKeyword)
	}
}

func TestServicePlanLLMRequestIncludesToolDescriptions(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentKnowledgeQA, []llmPlanStep{
		{Kind: StepKindRetrieve, Name: "retrieve"},
		{Kind: StepKindSynthesize, Name: "compose"},
		{Kind: StepKindCritic, Name: "validate"},
	})

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	_, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-tools-desc",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "how do I check tickets?",
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_search", ReadOnly: true},
			{Name: "ticket_comment_create", ReadOnly: false, RequiresApproval: true},
		},
		TenantPolicy: TenantPolicy{AllowToolUse: true},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if provider.captured == nil {
		t.Fatal("LLM provider was not called")
	}
	userMsg := provider.captured.Messages[0].Content
	if provider.captured.Messages[0].Role != "user" {
		t.Fatalf("first message role = %q, want %q", provider.captured.Messages[0].Role, "user")
	}
	// Verify tool descriptions are in the user message
	for _, toolName := range []string{"ticket_search", "ticket_comment_create"} {
		if !contains(userMsg, toolName) {
			t.Fatalf("user message does not contain tool %q", toolName)
		}
	}
	if !contains(userMsg, "allow_tool_use=true") {
		t.Fatal("user message does not contain tenant policy")
	}
}

func TestServicePlanWithLLMApprovalTool(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentIncidentAssist, []llmPlanStep{
		{Kind: StepKindTool, Name: "create comment", ToolName: "ticket_comment_create", ReadOnly: false, NeedsApproval: true},
		{Kind: StepKindSynthesize, Name: "compose", DependsOn: []string{"create comment"}},
		{Kind: StepKindCritic, Name: "validate", DependsOn: []string{"compose"}},
	})

	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-llm-approval",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "add a comment to ticket INC-200",
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_comment_create", ReadOnly: false, RequiresApproval: true},
		},
		TenantPolicy: TenantPolicy{AllowToolUse: true},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !got.RequiresTool {
		t.Fatal("RequiresTool = false, want true")
	}
	if !got.RequiresApproval {
		t.Fatal("RequiresApproval = false, want true")
	}
	assertToolStep(t, got.Steps, "ticket_comment_create", false)
}

func TestServicePlanWithLLMSendsTemperatureZero(t *testing.T) {
	planJSON := buildMockPlanJSON(IntentKnowledgeQA, []llmPlanStep{
		{Kind: StepKindRetrieve, Name: "retrieve"},
		{Kind: StepKindSynthesize, Name: "compose"},
		{Kind: StepKindCritic, Name: "validate"},
	})
	provider := &mockLLMProvider{
		response: llm.CompletionResponse{Content: planJSON, Model: "test-model"},
	}
	svc := NewServiceWithLLM(provider)

	_, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-temp",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if provider.captured == nil {
		t.Fatal("LLM provider was not called")
	}
	if provider.captured.Temperature == nil {
		t.Fatal("Temperature is nil, want explicit zero")
	}
	if *provider.captured.Temperature != 0 {
		t.Fatalf("Temperature = %f, want 0", *provider.captured.Temperature)
	}
}

// --- Prompt construction tests ---

func TestBuildPlannerUserMessageContainsAllInputs(t *testing.T) {
	msg := buildPlannerUserMessage(PlanInput{
		Mode:        "chat",
		UserMessage: "test query",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{
				{Kind: contextengine.BlockKindRecentTurns, Content: "user: hello"},
			},
		},
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_search", ReadOnly: true},
		},
		TenantPolicy: TenantPolicy{AllowToolUse: true},
	})

	for _, want := range []string{"chat", "test query", "recent_turns", "ticket_search", "allow_tool_use=true"} {
		if !contains(msg, want) {
			t.Fatalf("user message missing %q; got:\n%s", want, msg)
		}
	}
}

// --- Validation tests ---

func TestValidateLLMPlanRejectsInvalidKind(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps:  []llmPlanStep{{Kind: "invalid_kind", Name: "test"}},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for invalid kind")
	}
}

func TestValidateLLMPlanRejectsEmptySteps(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps:  []llmPlanStep{},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for empty steps")
	}
}

func TestValidateLLMPlanRejectsInvalidIntent(t *testing.T) {
	resp := llmPlanResponse{
		Intent: "bogus",
		Steps:  []llmPlanStep{{Kind: StepKindRetrieve, Name: "test"}},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for invalid intent")
	}
}

func TestValidateLLMPlanRejectsDuplicateStepNames(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps: []llmPlanStep{
			{Kind: StepKindRetrieve, Name: "retrieve"},
			{Kind: StepKindSynthesize, Name: "retrieve"}, // duplicate
		},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for duplicate step names")
	}
}

func TestValidateLLMPlanRejectsDanglingDependsOn(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps: []llmPlanStep{
			{Kind: StepKindRetrieve, Name: "retrieve"},
			{Kind: StepKindSynthesize, Name: "compose", DependsOn: []string{"nonexistent"}},
		},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for dangling depends_on")
	}
}

func TestValidateLLMPlanRejectsEmptyStepName(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps: []llmPlanStep{
			{Kind: StepKindRetrieve, Name: ""},
		},
	}
	err := validateLLMPlan(resp, nil)
	if err == nil {
		t.Fatal("validateLLMPlan() error = nil, want error for empty step name")
	}
}

func TestValidateLLMPlanAcceptsValidPlan(t *testing.T) {
	resp := llmPlanResponse{
		Intent: IntentKnowledgeQA,
		Steps: []llmPlanStep{
			{Kind: StepKindRetrieve, Name: "retrieve"},
			{Kind: StepKindSynthesize, Name: "synthesize"},
			{Kind: StepKindCritic, Name: "critic"},
		},
	}
	if err := validateLLMPlan(resp, nil); err != nil {
		t.Fatalf("validateLLMPlan() error = %v, want nil", err)
	}
}

// --- Strip code fences tests ---

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{"```json\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"```\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"  ```json\n{}\n```  ", `{}`},
	}
	for _, tt := range tests {
		got := stripCodeFences(tt.input)
		if got != tt.want {
			t.Errorf("stripCodeFences(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertStepKinds(t *testing.T, steps []PlanStep, wantKinds ...string) {
	t.Helper()

	if len(steps) != len(wantKinds) {
		t.Fatalf("len(steps) = %d, want %d", len(steps), len(wantKinds))
	}
	for i, wantKind := range wantKinds {
		if steps[i].Kind != wantKind {
			t.Fatalf("steps[%d].Kind = %q, want %q", i, steps[i].Kind, wantKind)
		}
	}
}

func assertToolStep(t *testing.T, steps []PlanStep, wantTool string, wantReadOnly bool) {
	t.Helper()

	for _, step := range steps {
		if step.Kind != StepKindTool {
			continue
		}
		if step.ToolName != wantTool {
			t.Fatalf("ToolName = %q, want %q", step.ToolName, wantTool)
		}
		if step.ReadOnly != wantReadOnly {
			t.Fatalf("ReadOnly = %v, want %v", step.ReadOnly, wantReadOnly)
		}
		return
	}

	t.Fatalf("tool step for %q not found in %#v", wantTool, steps)
}

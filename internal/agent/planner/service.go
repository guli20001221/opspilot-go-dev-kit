package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"opspilot-go/internal/llm"
)

const maxSteps = 6

// Service builds execution plans from runtime inputs.
// When an LLM provider is configured, it uses structured LLM output for intent
// classification and step planning. When no provider is available (or the LLM
// call fails), it falls back to deterministic keyword-based planning.
type Service struct {
	llm llm.Provider
}

// NewService constructs the planner service with deterministic keyword-based planning.
func NewService() *Service {
	return &Service{}
}

// NewServiceWithLLM constructs the planner service with an LLM provider for
// structured plan generation. If provider is nil or a PlaceholderProvider,
// the service falls back to keyword-based planning.
func NewServiceWithLLM(provider llm.Provider) *Service {
	if provider != nil {
		if _, isPlaceholder := provider.(*llm.PlaceholderProvider); isPlaceholder {
			provider = nil
		}
	}
	return &Service{llm: provider}
}

// Plan derives a structured execution plan from the request and context snapshot.
// When an LLM provider is available, it sends the request context to the LLM and
// parses the structured JSON response. On failure, it falls back to the
// deterministic keyword-based planner.
func (s *Service) Plan(ctx context.Context, input PlanInput) (ExecutionPlan, error) {
	planID := derivePlanID(input)

	if s.llm != nil {
		plan, err := s.planWithLLM(ctx, planID, input)
		if err != nil {
			slog.Warn("llm planner failed, falling back to keyword planner",
				slog.String("plan_id", planID),
				slog.Any("error", err),
			)
		} else {
			return plan, nil
		}
	}

	return s.planWithKeywords(input, planID), nil
}

// planWithLLM sends the planning request to the LLM and parses the structured response.
func (s *Service) planWithLLM(ctx context.Context, planID string, input PlanInput) (ExecutionPlan, error) {
	userMsg := buildPlannerUserMessage(input)

	resp, err := s.llm.Complete(ctx, llm.CompletionRequest{
		SystemPrompt:   plannerSystemPrompt,
		Messages:       []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:      1024,
		Temperature:    llm.TemperaturePtr(0),
		ResponseFormat: llm.ResponseFormatJSON,
	})
	if err != nil {
		return ExecutionPlan{}, fmt.Errorf("llm complete: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	// Strip markdown code fences if the model wraps its output.
	content = stripCodeFences(content)

	var planResp llmPlanResponse
	if err := json.Unmarshal([]byte(content), &planResp); err != nil {
		return ExecutionPlan{}, fmt.Errorf("unmarshal plan response: %w (raw: %s)", err, truncate(content, 200))
	}

	availableTools := make(map[string]bool, len(input.AvailableTools))
	for _, tool := range input.AvailableTools {
		availableTools[tool.Name] = true
	}
	if err := validateLLMPlan(planResp, availableTools); err != nil {
		return ExecutionPlan{}, fmt.Errorf("validate plan: %w", err)
	}

	plan := toLLMPlanResponse(planID, planResp)
	plan.Source = PlanSourceLLM
	plan.PromptVersion = PromptVersion

	slog.Info("llm planner produced plan",
		slog.String("plan_id", planID),
		slog.String("intent", plan.Intent),
		slog.Int("steps", len(plan.Steps)),
		slog.String("reasoning", plan.PlannerReasoningShort),
		slog.String("prompt_version", PromptVersion),
	)

	return plan, nil
}

// planWithKeywords is the deterministic fallback planner using keyword matching.
func (s *Service) planWithKeywords(input PlanInput, planID string) ExecutionPlan {
	intent := classifyIntent(input)
	requiresWorkflow := shouldPromoteWorkflow(input, intent)
	tool := selectTool(input)
	requiresTool := tool.Name != ""
	requiresApproval := tool.RequiresApproval
	requiresRetrieval := !requiresWorkflow
	if requiresTool && tool.ReadOnly {
		requiresRetrieval = false
	}

	plan := ExecutionPlan{
		PlanID:                planID,
		Intent:                intent,
		RequiresRetrieval:     requiresRetrieval,
		RequiresTool:          requiresTool,
		RequiresWorkflow:      requiresWorkflow || tool.AsyncOnly,
		RequiresApproval:      requiresApproval,
		OutputSchema:          selectOutputSchema(intent),
		PlannerReasoningShort: summarizeReasoning(intent, requiresTool, requiresWorkflow || tool.AsyncOnly),
		Source:                PlanSourceKeyword,
	}
	plan.Steps = buildSteps(plan, tool)
	plan.MaxSteps = len(plan.Steps)
	if plan.MaxSteps > maxSteps {
		plan.MaxSteps = maxSteps
		plan.Steps = plan.Steps[:maxSteps]
	}

	return plan
}

func classifyIntent(input PlanInput) string {
	message := strings.ToLower(input.UserMessage)
	switch {
	case input.Mode == "task":
		return IntentReportRequest
	case strings.Contains(message, "report"), strings.Contains(message, "export"):
		return IntentReportRequest
	case strings.Contains(message, "ticket"):
		return IntentIncidentAssist
	default:
		return IntentKnowledgeQA
	}
}

func shouldPromoteWorkflow(input PlanInput, intent string) bool {
	if input.Mode == "task" {
		return true
	}
	if intent == IntentReportRequest {
		return true
	}

	return false
}

func selectTool(input PlanInput) ToolDescriptor {
	if !input.TenantPolicy.AllowToolUse && input.TenantPolicy != (TenantPolicy{}) {
		return ToolDescriptor{}
	}

	message := strings.ToLower(input.UserMessage)
	if strings.Contains(message, "ticket") && (strings.Contains(message, "search") || strings.Contains(message, "history") || strings.Contains(message, "query")) {
		for _, tool := range input.AvailableTools {
			if tool.Name == "ticket_search" {
				return tool
			}
		}
	}

	for _, tool := range input.AvailableTools {
		if strings.Contains(message, "ticket") && strings.Contains(tool.Name, "ticket") {
			return tool
		}
	}

	return ToolDescriptor{}
}

func buildSteps(plan ExecutionPlan, tool ToolDescriptor) []PlanStep {
	if plan.RequiresWorkflow {
		return []PlanStep{
			{
				StepID:   "step-1",
				Kind:     StepKindPromoteWorkflow,
				Name:     "promote to workflow",
				ToolName: tool.Name,
			},
		}
	}

	steps := []PlanStep{
		{
			StepID: "step-1",
			Kind:   StepKindRetrieve,
			Name:   "retrieve supporting evidence",
		},
	}
	if tool.Name != "" {
		steps = append(steps, PlanStep{
			StepID:        "step-2",
			Kind:          StepKindTool,
			Name:          fmt.Sprintf("run %s", tool.Name),
			DependsOn:     []string{"step-1"},
			ToolName:      tool.Name,
			ReadOnly:      tool.ReadOnly,
			NeedsApproval: tool.RequiresApproval,
		})
	}

	dependsOn := []string{steps[len(steps)-1].StepID}
	steps = append(steps,
		PlanStep{
			StepID:    fmt.Sprintf("step-%d", len(steps)+1),
			Kind:      StepKindSynthesize,
			Name:      "compose grounded answer",
			DependsOn: dependsOn,
		},
		PlanStep{
			StepID:    fmt.Sprintf("step-%d", len(steps)+2),
			Kind:      StepKindCritic,
			Name:      "validate response",
			DependsOn: []string{fmt.Sprintf("step-%d", len(steps)+1)},
		},
	)

	return steps
}

func selectOutputSchema(intent string) string {
	if intent == IntentReportRequest {
		return "structured_summary"
	}

	return "markdown"
}

func summarizeReasoning(intent string, requiresTool bool, requiresWorkflow bool) string {
	parts := []string{intent}
	if requiresTool {
		parts = append(parts, "tool")
	}
	if requiresWorkflow {
		parts = append(parts, "workflow")
	}

	return strings.Join(parts, ", ")
}

func derivePlanID(input PlanInput) string {
	if input.RequestID != "" {
		return "plan-" + input.RequestID
	}
	if input.SessionID != "" {
		return "plan-" + input.SessionID
	}

	return "plan-generated"
}

// stripCodeFences removes markdown code fences that some models wrap around JSON.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// truncate returns the first n bytes of s, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

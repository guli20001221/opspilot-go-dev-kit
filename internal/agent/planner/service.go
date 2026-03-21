package planner

import (
	"context"
	"fmt"
	"strings"
)

const maxSteps = 6

// Service builds deterministic execution plans from runtime inputs.
type Service struct{}

// NewService constructs the planner service.
func NewService() *Service {
	return &Service{}
}

// Plan derives a structured execution plan from the request and context snapshot.
func (s *Service) Plan(_ context.Context, input PlanInput) (ExecutionPlan, error) {
	intent := classifyIntent(input)
	requiresWorkflow := shouldPromoteWorkflow(input, intent)
	tool := selectTool(input)
	requiresTool := tool.Name != ""
	requiresApproval := tool.RequiresApproval

	plan := ExecutionPlan{
		PlanID:                derivePlanID(input),
		Intent:                intent,
		RequiresRetrieval:     !requiresWorkflow,
		RequiresTool:          requiresTool,
		RequiresWorkflow:      requiresWorkflow || tool.AsyncOnly,
		RequiresApproval:      requiresApproval,
		OutputSchema:          selectOutputSchema(intent),
		PlannerReasoningShort: summarizeReasoning(intent, requiresTool, requiresWorkflow || tool.AsyncOnly),
	}
	plan.Steps = buildSteps(plan, tool)
	plan.MaxSteps = len(plan.Steps)
	if plan.MaxSteps > maxSteps {
		plan.MaxSteps = maxSteps
		plan.Steps = plan.Steps[:maxSteps]
	}

	return plan, nil
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

package planner

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// PromptVersion identifies the current planner prompt revision.
const PromptVersion = "planner-v1"

// plannerSystemPrompt is the LLM system prompt for structured plan generation.
// This prompt is versioned alongside the eval prompt file eval/prompts/planner-v1.md.
const plannerSystemPrompt = `You are the planning module of an enterprise operations assistant called OpsPilot.
Your job is to analyze the user's request and produce a structured execution plan as a JSON object.
You do NOT execute the plan — you only decide what should happen.

Return a single JSON object with this exact schema:
{
  "intent": "<knowledge_qa|incident_assist|report_request>",
  "reasoning": "<1-2 sentence explanation of why you chose this plan>",
  "requires_retrieval": <true|false>,
  "requires_tool": <true|false>,
  "requires_workflow": <true|false>,
  "requires_approval": <true|false>,
  "output_schema": "<markdown|structured_summary>",
  "steps": [
    {
      "kind": "<retrieve|tool|synthesize|critic|promote_workflow>",
      "name": "<human-readable step name>",
      "depends_on": ["<step names this depends on>"],
      "tool_name": "<tool name if kind=tool, empty otherwise>",
      "read_only": <true|false>,
      "needs_approval": <true|false>
    }
  ]
}

Intent classification:
- "knowledge_qa": Question answerable from knowledge base. Default.
- "incident_assist": About incidents, tickets, or ops issues that may need tool lookups.
- "report_request": Report, export, or long-running analysis. Always requires_workflow=true.

Planning rules:
1. If mode is "task", set intent to "report_request" and requires_workflow to true.
2. Report/export requests: intent "report_request", requires_workflow true.
3. If requires_workflow is true, produce exactly one step: promote_workflow.
4. If user mentions tickets and a ticket tool is available, include a tool step.
5. Read-only tool steps do NOT require approval.
6. Write tool steps (read_only=false) REQUIRE approval.
7. If a tool has async_only=true, set requires_workflow to true.
8. Non-workflow plans end with synthesize then critic steps.
9. If retrieval is needed, start with a retrieve step.
10. Retrieve before tool if both are needed.
11. Do not use tools if tenant policy disallows tool use.
12. Maximum 6 steps.
13. output_schema: "structured_summary" for report_request, "markdown" otherwise.

Output ONLY the JSON object. No markdown fences. No extra text. Every field is required.`

// buildPlannerUserMessage constructs the user message for the planning LLM call.
func buildPlannerUserMessage(input PlanInput) string {
	var b strings.Builder

	b.WriteString("Request mode: ")
	b.WriteString(input.Mode)
	b.WriteString("\n\nUser message: ")
	b.WriteString(input.UserMessage)

	if len(input.Context.Blocks) > 0 {
		b.WriteString("\n\nContext blocks:\n")
		for _, block := range input.Context.Blocks {
			fmt.Fprintf(&b, "- [%s] %s\n", block.Kind, block.Content)
		}
	}

	if len(input.AvailableTools) > 0 {
		b.WriteString("\nAvailable tools:\n")
		for _, tool := range input.AvailableTools {
			data, _ := json.Marshal(map[string]any{
				"name":              tool.Name,
				"read_only":         tool.ReadOnly,
				"requires_approval": tool.RequiresApproval,
				"async_only":        tool.AsyncOnly,
			})
			b.Write(data)
			b.WriteByte('\n')
		}
	}

	fmt.Fprintf(&b, "\nTenant policy: allow_tool_use=%v", input.TenantPolicy.AllowToolUse)

	return b.String()
}

// llmPlanResponse is the JSON schema the LLM must return.
type llmPlanResponse struct {
	Intent            string        `json:"intent"`
	Reasoning         string        `json:"reasoning"`
	RequiresRetrieval bool          `json:"requires_retrieval"`
	RequiresTool      bool          `json:"requires_tool"`
	RequiresWorkflow  bool          `json:"requires_workflow"`
	RequiresApproval  bool          `json:"requires_approval"`
	OutputSchema      string        `json:"output_schema"`
	Steps             []llmPlanStep `json:"steps"`
}

// llmPlanStep is one step in the LLM plan response.
type llmPlanStep struct {
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	DependsOn     []string `json:"depends_on"`
	ToolName      string   `json:"tool_name"`
	ReadOnly      bool     `json:"read_only"`
	NeedsApproval bool     `json:"needs_approval"`
}

// toLLMPlanResponse converts an llmPlanResponse into an ExecutionPlan.
func toLLMPlanResponse(planID string, resp llmPlanResponse) ExecutionPlan {
	steps := make([]PlanStep, 0, len(resp.Steps))
	for i, s := range resp.Steps {
		steps = append(steps, PlanStep{
			StepID:        fmt.Sprintf("step-%d", i+1),
			Kind:          s.Kind,
			Name:          s.Name,
			DependsOn:     s.DependsOn,
			ToolName:      s.ToolName,
			ReadOnly:      s.ReadOnly,
			NeedsApproval: s.NeedsApproval,
		})
	}

	maxStepsCount := len(steps)
	if maxStepsCount > maxSteps {
		slog.Warn("llm plan exceeds max steps, truncating",
			slog.String("plan_id", planID),
			slog.Int("original_steps", maxStepsCount),
			slog.Int("max_steps", maxSteps),
		)
		maxStepsCount = maxSteps
		steps = steps[:maxSteps]
	}

	return ExecutionPlan{
		PlanID:                planID,
		Intent:                resp.Intent,
		RequiresRetrieval:     resp.RequiresRetrieval,
		RequiresTool:          resp.RequiresTool,
		RequiresWorkflow:      resp.RequiresWorkflow,
		RequiresApproval:      resp.RequiresApproval,
		MaxSteps:              maxStepsCount,
		OutputSchema:          resp.OutputSchema,
		Steps:                 steps,
		PlannerReasoningShort: resp.Reasoning,
	}
}

// validateLLMPlan performs structural and policy validation on the parsed LLM plan.
// availableTools is the set of tool names the planner is allowed to reference.
func validateLLMPlan(resp llmPlanResponse, availableTools map[string]bool) error {
	validIntents := map[string]bool{
		IntentKnowledgeQA:    true,
		IntentIncidentAssist: true,
		IntentReportRequest:  true,
	}
	if !validIntents[resp.Intent] {
		return fmt.Errorf("invalid intent %q", resp.Intent)
	}

	if len(resp.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}

	validKinds := map[string]bool{
		StepKindRetrieve:        true,
		StepKindTool:            true,
		StepKindSynthesize:      true,
		StepKindCritic:          true,
		StepKindPromoteWorkflow: true,
	}

	nameSet := make(map[string]bool, len(resp.Steps))
	for i, step := range resp.Steps {
		if !validKinds[step.Kind] {
			return fmt.Errorf("step %d has invalid kind %q", i, step.Kind)
		}
		if step.Kind == StepKindTool && step.ToolName != "" && len(availableTools) > 0 {
			if !availableTools[step.ToolName] {
				return fmt.Errorf("step %d references unknown tool %q", i, step.ToolName)
			}
		}
		if step.Name == "" {
			return fmt.Errorf("step %d has empty name", i)
		}
		if nameSet[step.Name] {
			return fmt.Errorf("step %d has duplicate name %q", i, step.Name)
		}
		nameSet[step.Name] = true
	}

	// Validate depends_on references point to names that exist earlier in the plan.
	for i, step := range resp.Steps {
		for _, dep := range step.DependsOn {
			if !nameSet[dep] {
				return fmt.Errorf("step %d (%q) depends on unknown step %q", i, step.Name, dep)
			}
		}
	}

	return nil
}

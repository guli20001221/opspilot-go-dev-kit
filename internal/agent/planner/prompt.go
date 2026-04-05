package planner

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// PromptVersion identifies the current planner prompt revision.
const PromptVersion = "planner-v2"

// plannerSystemPrompt is the LLM system prompt for structured plan generation.
// This prompt is versioned alongside the eval prompt file eval/prompts/planner-v2.md.
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
      "tool_arguments": {<structured parameters for the tool, based on parameter schema>},
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
12. Maximum 6 steps (or fewer if tenant policy specifies a lower max_steps).
13. output_schema: "structured_summary" for report_request, "markdown" otherwise.
14. If the tenant has an allowed_tools list, only use tools from that list.
15. If the tenant has a forbidden_tools list, never include those tools in the plan.
16. If the tenant requires approval for writes, set needs_approval=true on every write (non-read-only) tool step.
17. If a tool has async_only=true, set requires_workflow=true and use promote_workflow.
18. A tool step's read_only and needs_approval MUST match the tool's registry attributes — do not mark a write tool as read_only.
19. For tool steps, populate tool_arguments with a JSON object matching the tool's parameter schema. Extract values from the user message and context. Use {} for non-tool steps.

Output ONLY the JSON object. No markdown fences. No extra text. Every field is required.`

// replanSystemPrompt is the LLM system prompt for dynamic replanning after partial execution.
const replanSystemPrompt = `You are the replanning module of OpsPilot, an enterprise operations assistant.
A previous execution plan partially executed but encountered issues. Your job is to produce a REVISED plan
that accounts for what already happened and adjusts the remaining steps.

You receive:
- The original plan and its intent
- Steps that already executed with their outcomes (succeeded, failed, approval_required)
- The reason replanning was triggered
- The user's original message and available tools

Return a single JSON object with the SAME schema as the original planner:
{
  "intent": "<knowledge_qa|incident_assist|report_request>",
  "reasoning": "<1-2 sentence explanation of how you adapted the plan>",
  "requires_retrieval": <true|false>,
  "requires_tool": <true|false>,
  "requires_workflow": <true|false>,
  "requires_approval": <true|false>,
  "output_schema": "<markdown|structured_summary>",
  "steps": [...]
}

Replanning rules:
1. Do NOT repeat steps that already succeeded — their results are already available.
2. If a tool step failed, decide whether to retry with different arguments, skip it, or replace it with an alternative.
3. If a tool step required approval, do not re-include it — it will be handled via the approval workflow.
4. Always end non-workflow plans with synthesize then critic steps.
5. The synthesize step should incorporate results from previously succeeded steps.
6. If the failure makes the original intent unachievable, simplify to a knowledge_qa plan with retrieval.
7. Maximum 6 steps total (including steps from the original plan that are being replaced).
8. Preserve the original intent unless the failure fundamentally changes what's possible.

Output ONLY the JSON object. No markdown fences. No extra text.`

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
			entry := map[string]any{
				"name":              tool.Name,
				"read_only":         tool.ReadOnly,
				"requires_approval": tool.RequiresApproval,
				"async_only":        tool.AsyncOnly,
			}
			if tool.Description != "" {
				entry["description"] = tool.Description
			}
			if len(tool.Parameters) > 0 {
				params := make([]map[string]any, 0, len(tool.Parameters))
				for _, p := range tool.Parameters {
					params = append(params, map[string]any{
						"name":        p.Name,
						"type":        p.Type,
						"required":    p.Required,
						"description": p.Description,
					})
				}
				entry["parameters"] = params
			}
			data, _ := json.Marshal(entry)
			b.Write(data)
			b.WriteByte('\n')
		}
	}

	if input.TenantPolicy.Configured {
		fmt.Fprintf(&b, "\nTenant policy: allow_tool_use=%v", input.TenantPolicy.AllowToolUse)
		if len(input.TenantPolicy.AllowedTools) > 0 {
			fmt.Fprintf(&b, ", allowed_tools=[%s]", strings.Join(input.TenantPolicy.AllowedTools, ","))
		}
		if len(input.TenantPolicy.ForbiddenTools) > 0 {
			fmt.Fprintf(&b, ", forbidden_tools=[%s]", strings.Join(input.TenantPolicy.ForbiddenTools, ","))
		}
		if input.TenantPolicy.MaxSteps > 0 {
			fmt.Fprintf(&b, ", max_steps=%d", input.TenantPolicy.MaxSteps)
		}
		if input.TenantPolicy.RequireApprovalForWrite {
			b.WriteString(", require_approval_for_write=true")
		}
	} else {
		b.WriteString("\nTenant policy: not configured (permissive defaults)")
	}

	return b.String()
}

// llmPlanResponse is the JSON schema the LLM must return.
// buildReplanUserMessage constructs the user message for the replanning LLM call.
func buildReplanUserMessage(input ReplanInput) string {
	var b strings.Builder

	b.WriteString("Original intent: ")
	b.WriteString(input.OriginalPlan.Intent)
	b.WriteString("\nOriginal reasoning: ")
	b.WriteString(input.OriginalPlan.PlannerReasoningShort)

	b.WriteString("\n\nUser message: ")
	b.WriteString(input.Input.UserMessage)

	b.WriteString("\n\nReplan reason: ")
	b.WriteString(input.ReplanReason)

	if len(input.ExecutedSteps) > 0 {
		b.WriteString("\n\nAlready executed steps:\n")
		for _, step := range input.ExecutedSteps {
			fmt.Fprintf(&b, "- [%s] %s (tool=%s, status=%s): %s\n",
				step.Kind, step.StepID, step.ToolName, step.Status, step.Summary)
		}
	}

	if len(input.Input.AvailableTools) > 0 {
		b.WriteString("\nAvailable tools:\n")
		for _, tool := range input.Input.AvailableTools {
			entry := map[string]any{
				"name":              tool.Name,
				"read_only":         tool.ReadOnly,
				"requires_approval": tool.RequiresApproval,
				"async_only":        tool.AsyncOnly,
			}
			if tool.Description != "" {
				entry["description"] = tool.Description
			}
			if len(tool.Parameters) > 0 {
				params := make([]map[string]any, 0, len(tool.Parameters))
				for _, p := range tool.Parameters {
					params = append(params, map[string]any{
						"name":        p.Name,
						"type":        p.Type,
						"required":    p.Required,
						"description": p.Description,
					})
				}
				entry["parameters"] = params
			}
			data, _ := json.Marshal(entry)
			b.Write(data)
			b.WriteByte('\n')
		}
	}

	if input.Input.TenantPolicy.Configured {
		fmt.Fprintf(&b, "\nTenant policy: allow_tool_use=%v", input.Input.TenantPolicy.AllowToolUse)
	}

	return b.String()
}

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
	Kind          string          `json:"kind"`
	Name          string          `json:"name"`
	DependsOn     []string        `json:"depends_on"`
	ToolName      string          `json:"tool_name"`
	ToolArguments json.RawMessage `json:"tool_arguments"`
	ReadOnly      bool            `json:"read_only"`
	NeedsApproval bool            `json:"needs_approval"`
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
			ToolArguments: s.ToolArguments,
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
		if step.Kind == StepKindTool {
			if step.ToolName == "" {
				return fmt.Errorf("step %d is kind %q but has empty tool_name", i, StepKindTool)
			}
			if len(availableTools) > 0 && !availableTools[step.ToolName] {
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

	// Workflow-shape invariant: if requires_workflow is true, the plan must
	// collapse to exactly one promote_workflow step with no executable steps
	// (tool, retrieve, synthesize, critic). This prevents synchronous side
	// effects before async workflow promotion.
	if resp.RequiresWorkflow {
		if len(resp.Steps) != 1 || resp.Steps[0].Kind != StepKindPromoteWorkflow {
			return fmt.Errorf("requires_workflow is true but plan has %d steps (expected exactly 1 promote_workflow step)", len(resp.Steps))
		}
	}

	// Converse invariant: promote_workflow steps may only appear when
	// requires_workflow is true.
	if !resp.RequiresWorkflow {
		for i, step := range resp.Steps {
			if step.Kind == StepKindPromoteWorkflow {
				return fmt.Errorf("step %d is promote_workflow but requires_workflow is false", i)
			}
		}
	}

	return nil
}

// validatePlanPolicy enforces tenant-level and registry-level policy on a parsed plan.
// tools maps tool names to their full descriptors from the registry.
// Registry invariants (async-only, approval, read-only classification) are always enforced.
// Tenant-specific constraints (allowed/forbidden, max steps) are only enforced when policy.Configured is true.
func validatePlanPolicy(resp llmPlanResponse, tools map[string]ToolDescriptor, policy TenantPolicy) error {
	// --- Tenant-level constraints (only when explicitly configured) ---
	if policy.Configured {
		if !policy.AllowToolUse {
			for i, step := range resp.Steps {
				if step.Kind == StepKindTool {
					return fmt.Errorf("step %d uses tool %q but tenant policy disallows tool use", i, step.ToolName)
				}
			}
		}

		effectiveMax := maxSteps
		if policy.MaxSteps > 0 && policy.MaxSteps < effectiveMax {
			effectiveMax = policy.MaxSteps
		}
		if len(resp.Steps) > effectiveMax {
			return fmt.Errorf("plan has %d steps, tenant policy allows at most %d", len(resp.Steps), effectiveMax)
		}

		forbiddenSet := make(map[string]bool, len(policy.ForbiddenTools))
		for _, t := range policy.ForbiddenTools {
			forbiddenSet[t] = true
		}
		allowedSet := make(map[string]bool, len(policy.AllowedTools))
		for _, t := range policy.AllowedTools {
			allowedSet[t] = true
		}

		for i, step := range resp.Steps {
			if step.Kind != StepKindTool {
				continue
			}
			if forbiddenSet[step.ToolName] {
				return fmt.Errorf("step %d uses forbidden tool %q", i, step.ToolName)
			}
			if len(allowedSet) > 0 && !allowedSet[step.ToolName] {
				return fmt.Errorf("step %d uses tool %q not in tenant allowed list", i, step.ToolName)
			}
			if policy.RequireApprovalForWrite {
				desc, ok := tools[step.ToolName]
				if ok && !desc.ReadOnly && !step.NeedsApproval {
					return fmt.Errorf("step %d uses write tool %q but tenant policy requires approval for all writes", i, step.ToolName)
				}
			}
		}
	}

	// --- Registry invariants (always enforced) ---
	for i, step := range resp.Steps {
		if step.Kind != StepKindTool {
			continue
		}
		desc, ok := tools[step.ToolName]
		if !ok {
			continue
		}
		if desc.AsyncOnly && !resp.RequiresWorkflow {
			return fmt.Errorf("step %d uses async-only tool %q but plan does not set requires_workflow", i, step.ToolName)
		}
		if !desc.ReadOnly && desc.RequiresApproval && !step.NeedsApproval {
			return fmt.Errorf("step %d uses tool %q which requires approval, but needs_approval is false", i, step.ToolName)
		}
		if !desc.ReadOnly && step.ReadOnly {
			return fmt.Errorf("step %d marks tool %q as read_only but registry classifies it as write", i, step.ToolName)
		}
	}

	return nil
}

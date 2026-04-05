// Package planner contains the typed planning stage for the agent runtime.
//
// When an LLM provider is configured, the planner sends the user request,
// conversation context, available tools, and tenant policy to the LLM and
// parses a structured JSON execution plan. This enables intent classification,
// tool selection, and step sequencing to leverage LLM reasoning rather than
// keyword heuristics.
//
// When no LLM provider is available (or the LLM call fails), the planner
// falls back to deterministic keyword-based planning for reliability.
//
// The planner prompt is versioned as code (eval/prompts/planner-v2.md) and
// the prompt version is recorded with every plan for reproducibility.
package planner

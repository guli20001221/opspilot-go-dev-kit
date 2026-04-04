# Critic Prompt v1

Evaluates draft answers for quality, groundedness, and safety.

## Dimensions
- **Groundedness** (0-1): Are claims supported by provided context?
- **Citation Coverage** (0-1): Are evidence sources properly referenced?
- **Tool Consistency** (0-1): Do tool results match the answer?
- **Risk Level**: low / medium / high

## Verdicts
- `approve`: Ready to send
- `revise`: Needs improvement (with hints)
- `promote_workflow`: Requires async human review
- `reject`: Should be blocked (with reasons)

## Output Schema
```json
{
  "verdict": "approve|revise|promote_workflow|reject",
  "groundedness": 0.0-1.0,
  "citation_coverage": 0.0-1.0,
  "tool_consistency": 0.0-1.0,
  "risk_level": "low|medium|high",
  "missing_items": [],
  "revision_hints": [],
  "blocking_reasons": [],
  "reasoning": "brief explanation"
}
```

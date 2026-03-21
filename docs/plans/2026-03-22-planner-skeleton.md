# Planner Skeleton Plan

**Goal:** add a deterministic typed planner under `internal/agent/planner` and wire it into the synchronous chat application flow after context assembly.

**Scope:**
- define `PlanInput`, `ExecutionPlan`, `PlanStep`, and tool descriptors
- classify the current request into a small intent set
- derive `RequiresRetrieval`, `RequiresTool`, `RequiresWorkflow`, and `RequiresApproval`
- emit deterministic step lists with `retrieve`, `tool`, `synthesize`, `critic`, and `promote_workflow`
- expose the generated plan on the internal chat application result

**Out of scope:**
- prompt-backed planning
- retrieval execution
- tool execution
- critic execution
- exposing plan details over the public SSE API

**Verification:**
- `go test ./internal/agent/planner -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`

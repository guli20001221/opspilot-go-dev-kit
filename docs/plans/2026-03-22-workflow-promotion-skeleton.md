# Workflow Promotion Skeleton Plan

**Goal:** add a typed in-memory workflow promotion service and wire the synchronous chat application flow to create promoted tasks when the planner or critic requires async execution.

**Scope:**
- define `PromoteRequest`, `Task`, and initial task statuses
- create promoted tasks in `queued` or `waiting_approval`
- expose promoted task results on the internal chat application result
- keep the public HTTP/SSE contract unchanged for now

**Out of scope:**
- Temporal workflows
- task status APIs
- public `task_promoted` SSE events
- approval resume paths

**Verification:**
- `go test ./internal/workflow -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`

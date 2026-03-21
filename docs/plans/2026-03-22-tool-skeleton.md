# Tool Skeleton Plan

**Goal:** add a typed tool-execution stage and a minimal tool registry, then wire read-only tool execution into the synchronous chat application flow when the planner emits tool steps.

**Scope:**
- define `ToolInvocation` and `ToolResult`
- add an in-process tool registry under `internal/tools/registry`
- execute deterministic stub tools without live external calls
- gate write/admin tools behind `approval_required`
- expose tool results on the internal chat application result

**Out of scope:**
- real ticket system integration
- persistence to `tool_calls`
- workflow resume after approval
- public SSE `tool` events

**Verification:**
- `go test ./internal/agent/tool -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`

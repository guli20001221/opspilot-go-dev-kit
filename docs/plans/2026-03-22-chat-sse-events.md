# Chat SSE Event Expansion Plan

**Goal:** expand the synchronous chat SSE stream to expose internal planner, retrieval, tool, and async-promotion milestones while keeping the existing `meta`, `state`, and `done` events compatible.

**Scope:**
- emit a `plan` event for every chat request
- emit a `retrieval` event when retrieval runs
- emit a `tool` event for each tool execution
- emit a `task_promoted` event when the internal workflow layer creates a promoted task
- keep the public endpoint and error envelope unchanged

**Out of scope:**
- task status APIs
- public SSE `citation` events
- token streaming deltas
- approval resume APIs

**Verification:**
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

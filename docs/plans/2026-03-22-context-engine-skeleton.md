# Context Engine Skeleton Plan

**Goal:** add a deterministic `internal/contextengine` package that assembles typed context blocks and a `ContextAssemblyLog`, then wire it into the current synchronous chat application service.

**Scope:**
- create a minimal context engine package with typed inputs, outputs, and block kinds
- assemble `user_profile`, `recent_turns`, `session_summary`, and `task_scratchpad` blocks when data exists
- enforce an explicit assembly budget with deterministic low-priority dropping
- call the context engine from `internal/app/chat` before the placeholder assistant response is persisted

**Out of scope:**
- planner integration
- retrieval or memory hits
- prompt composition
- trace export of the assembly log

**Verification:**
- `go test ./internal/contextengine -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`

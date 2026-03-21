# M1 Chat Application Service Plan

**Goal:** move the Milestone 1 synchronous chat flow out of the HTTP handler into a typed application service so the transport boundary stays thin before Planner, Retrieval, Tool, and Critic runtime land.

**Scope:**
- create `internal/app/chat` with a typed `ChatRequestEnvelope`
- keep the existing in-memory `internal/session` dependency
- let the chat service persist user and assistant messages
- let the chat service assemble ordered `meta -> state -> done` SSE events
- keep the external HTTP contract unchanged

**Out of scope:**
- planner or retrieval integration
- database-backed session storage
- auth, RBAC, or tenant enforcement
- workflow promotion

**Verification:**
- `go test ./internal/app/chat`
- `go test ./internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- host-side POST to `http://localhost:18080/api/v1/chat/stream`

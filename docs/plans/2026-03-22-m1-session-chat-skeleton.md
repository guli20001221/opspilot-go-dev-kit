# M1 Session Chat Skeleton Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the smallest Milestone 1 slice: session/message CRUD in an in-memory service and a `POST /api/v1/chat/stream` SSE skeleton that persists user and assistant messages.

**Architecture:** Keep handlers thin behind `internal/session` and `internal/app/httpapi`. Use an in-memory repository first so the API contract and process wiring stabilize before introducing database-backed session storage. The chat stream endpoint emits only `meta`, `state`, and `done/error` events for now, with a fixed assistant placeholder response and persisted user/assistant messages.

**Tech Stack:** Go, standard library `net/http`, SSE over `text/event-stream`, in-memory storage with mutex protection.

---

### Task 1: Session service and in-memory repository

**Files:**
- Create: `internal/session/service.go`
- Create: `internal/session/types.go`
- Create: `internal/session/service_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- creating a session returns a generated session ID
- appending user and assistant messages persists ordering
- reading session messages returns the stored sequence

**Step 2: Run test to verify it fails**

Run: `go test ./internal/session`
Expected: FAIL because package does not exist yet

**Step 3: Write minimal implementation**

Implement a mutex-protected in-memory store and service methods for:
- create session
- append message
- list messages

**Step 4: Run test to verify it passes**

Run: `go test ./internal/session`
Expected: PASS

### Task 2: Session HTTP endpoints

**Files:**
- Modify: `internal/app/httpapi/health.go`
- Create: `internal/app/httpapi/sessions.go`
- Create: `internal/app/httpapi/sessions_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- `POST /api/v1/sessions` creates a session
- `GET /api/v1/sessions/{id}/messages` returns persisted messages
- unknown sessions return a uniform JSON error

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/httpapi`
Expected: FAIL because endpoints do not exist yet

**Step 3: Write minimal implementation**

Wire the session service into the handler tree and return JSON responses.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/httpapi`
Expected: PASS

### Task 3: Chat stream SSE skeleton

**Files:**
- Modify: `internal/app/httpapi/sessions.go`
- Create: `internal/app/httpapi/chat_stream_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- `POST /api/v1/chat/stream` returns `text/event-stream`
- event order is `meta`, `state`, `done`
- user and assistant messages are persisted in the session

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/httpapi`
Expected: FAIL because chat stream endpoint does not exist yet

**Step 3: Write minimal implementation**

Implement:
- request parsing for the Milestone 1 subset
- session creation when `session_id` is empty
- persistence of user and placeholder assistant messages
- SSE events `meta`, `state`, `done`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/httpapi`
Expected: PASS

### Task 4: Verification and docs sync

**Files:**
- Modify: `README.md`
- Modify: `docs/runbooks/local-bootstrap.md`

**Step 1: Run verification**

Run:
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

**Step 2: Update docs**

Document the new endpoints and note that session storage is in-memory for now.

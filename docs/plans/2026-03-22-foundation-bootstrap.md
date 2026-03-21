# Foundation Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the smallest runnable Foundation slice for OpsPilot-Go: module metadata, configuration, logging, API/worker entrypoints, health endpoints, Makefile targets, and bootstrap docs.

**Architecture:** Keep the first slice as a modular-monolith skeleton. The API binary serves `/healthz` and `/readyz` over `net/http` with request-scoped IDs and structured logs. The worker binary only boots config and logging, then waits for shutdown so later workflow work can land cleanly without rewriting process setup.

**Tech Stack:** Go, standard library `net/http`, `log/slog`, environment-based config, GNU Make.

---

### Task 1: Module metadata and shared config

**Files:**
- Create: `go.mod`
- Create: `internal/app/config/config.go`
- Test: `internal/app/config/config_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- default API and worker addresses are applied when env vars are absent
- env overrides are respected
- invalid required values return an error

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/config`
Expected: FAIL because package does not exist yet

**Step 3: Write minimal implementation**

Implement an environment loader that returns a typed `Config` with:
- `Env`
- `LogLevel`
- `APIListenAddr`
- `WorkerShutdownTimeout`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/config`
Expected: PASS

### Task 2: HTTP health handlers and request context

**Files:**
- Create: `internal/app/httpapi/health.go`
- Create: `internal/app/httpapi/middleware.go`
- Test: `internal/app/httpapi/health_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- `GET /healthz` returns `200` with a JSON status payload
- `GET /readyz` returns `200` with a JSON status payload
- middleware sets `X-Request-Id` on the response

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/httpapi`
Expected: FAIL because package does not exist yet

**Step 3: Write minimal implementation**

Implement:
- router construction
- health/readiness handlers
- request ID middleware
- trace ID extraction from request headers when present

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/httpapi`
Expected: PASS

### Task 3: API and worker entrypoints

**Files:**
- Create: `cmd/api/main.go`
- Create: `cmd/worker/main.go`
- Create: `internal/app/logging/logging.go`

**Step 1: Write the failing test**

Use package-level tests where practical for constructor-level behavior, then verify binaries compile.

**Step 2: Run compile check to verify it fails**

Run: `go test ./...`
Expected: FAIL until binaries and shared packages are wired

**Step 3: Write minimal implementation**

Implement:
- shared `slog` logger creation
- API process bootstrap and graceful shutdown
- worker process bootstrap and graceful shutdown

**Step 4: Run compile check to verify it passes**

Run: `go test ./...`
Expected: PASS

### Task 4: Developer workflow and docs

**Files:**
- Create: `Makefile`
- Create: `.env.example`
- Create: `docs/runbooks/local-bootstrap.md`
- Modify: `README.md`

**Step 1: Write the failing test**

For doc-and-build workflow work, use executable verification rather than unit tests.

**Step 2: Run verification to show gap**

Run:
- `make test`
- `make build`

Expected: commands unavailable before Makefile exists

**Step 3: Write minimal implementation**

Add:
- `fmt`
- `test`
- `build`
- `check`
- explicit placeholder `dev-up` / `dev-down` targets with clear failure guidance until compose stack lands

Document the actual commands used in README and the local bootstrap runbook.

**Step 4: Run verification**

Run:
- `make test`
- `make build`

Expected: PASS

### Task 5: Foundation verification

**Files:**
- No new files unless verification exposes gaps

**Step 1: Run full verification**

Run:
- `go test ./...`
- `make test`
- `make build`

**Step 2: Inspect outputs**

Confirm:
- all tests pass
- binaries compile
- docs reflect actual commands

**Step 3: Report remaining Foundation gaps**

Call out intentionally deferred work:
- DB migrations
- real dev stack
- Redis / Temporal wiring
- OpenTelemetry exporter setup

# Foundation Services Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the next Foundation slice: initial SQL migration scaffolding, a local Docker Compose stack, and developer commands to manage it.

**Architecture:** Keep the app runtime minimal while making local infrastructure explicit. PostgreSQL, Redis, and Temporal are started through `compose.yaml`; the API and worker also run in the same stack via bind-mounted source and `go run`, keeping the first environment reproducible without introducing custom container builds yet. A small Go migration package validates migration discovery and ordering so migration behavior remains typed and testable even before full DB repository work lands.

**Tech Stack:** Go, standard library, Docker Compose, PostgreSQL, Redis, Temporal auto-setup.

---

### Task 1: Migration planning package

**Files:**
- Create: `internal/storage/migrate/plan.go`
- Create: `internal/storage/migrate/plan_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- `.sql` migration files are discovered and sorted by filename
- non-SQL files are ignored
- missing directories return an error

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/migrate`
Expected: FAIL because package does not exist yet

**Step 3: Write minimal implementation**

Implement a typed migration descriptor and discovery function for a filesystem directory.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/migrate`
Expected: PASS

### Task 2: Initial SQL migration

**Files:**
- Create: `db/migrations/000001_foundation.sql`

**Step 1: Define the smallest schema slice**

Create additive SQL for:
- `tenants`
- `users`
- timestamps and tenant relationship

**Step 2: Verify migration discovery sees the file**

Run: `go test ./internal/storage/migrate`
Expected: PASS with the migration file included by discovery rules

### Task 3: Local Docker Compose stack

**Files:**
- Create: `compose.yaml`
- Modify: `.env.example`

**Step 1: Write configuration verification**

Use executable validation for the compose file rather than unit tests.

**Step 2: Run validation to show current gap**

Run: `docker compose config`
Expected: FAIL because `compose.yaml` does not exist yet

**Step 3: Write minimal implementation**

Add services for:
- `postgres`
- `redis`
- `temporal-postgresql`
- `temporal`
- `temporal-ui`
- `api`
- `worker`

Pin explicit ports and environment variables.

**Step 4: Run validation**

Run: `docker compose config`
Expected: PASS

### Task 4: Developer commands and runbook sync

**Files:**
- Modify: `Makefile`
- Modify: `scripts/dev/tasks.ps1`
- Modify: `docs/runbooks/local-bootstrap.md`
- Modify: `README.md`

**Step 1: Run current command gap**

Run:
- `make dev-up`
- `make dev-down`

Expected: placeholder failure

**Step 2: Write minimal implementation**

Replace placeholders with `docker compose up -d` and `docker compose down`.
Add matching PowerShell fallback commands.
Document the requirement that Docker Desktop must be running.

**Step 3: Run verification**

Run:
- `docker compose config`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 dev-down`

Expected:
- compose config passes
- dev-down succeeds even when nothing is running

### Task 5: Foundation services verification

**Files:**
- No new files unless verification exposes a gap

**Step 1: Run full verification**

Run:
- `go test ./...`
- `docker compose config`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

**Step 2: Report actual runtime gap**

If Docker daemon is unavailable, report that `dev-up` remains unverified in this environment instead of claiming success.

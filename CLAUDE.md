# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-specific notes
- Treat `AGENTS.md` as the canonical repository-wide rule set.
- Treat `.claude/skills/<skill-name>/SKILL.md` as the canonical task playbooks.
- If path-specific guidance grows large, move it into `.claude/rules/` or nearby `AGENTS.override.md` files instead of duplicating repository-wide instructions.

## Build, test, and dev commands

```bash
# Format
make fmt

# Run all tests
make test

# Run a single package's tests
go test ./internal/eval/...

# Run a single test by name
go test ./internal/eval/... -run TestRunService

# Build all binaries (api, worker, ticketapi → ./bin/)
make build

# Full check (fmt + test + build)
make check

# Local dev stack (Postgres, Redis, Temporal, API, Worker, fake TicketAPI)
make dev-up    # docker compose up -d --build
make dev-down  # docker compose down
```

PowerShell fallback when `make` is unavailable: `scripts/dev/tasks.ps1`.

## Local stack details

`compose.yaml` brings up: postgres (5432), redis (6379), temporal (7233), temporal-ui (8088), api (**18080**), worker, ticket-api (19090). The API is published on port 18080 to avoid 8080 conflicts. Postgres seeds migrations from `db/migrations/` on first run.

Default DSN: `postgres://opspilot:opspilot@localhost:5432/opspilot?sslmode=disable`

## Configuration

All config is env-var driven via `internal/app/config`. Every var is prefixed `OPSPILOT_`. Key vars:

| Variable | Default | Purpose |
|---|---|---|
| `OPSPILOT_POSTGRES_DSN` | localhost DSN | Primary database |
| `OPSPILOT_TEMPORAL_ENABLED` | `false` | Enable Temporal workflows |
| `OPSPILOT_TEMPORAL_ADDRESS` | `localhost:7233` | Temporal server |
| `OPSPILOT_EVAL_JUDGE_PROVIDER` | `placeholder` | `placeholder` or `http` for external judge |
| `OPSPILOT_TICKET_API_BASE_URL` | (empty) | Fake ticket API for local dev |
| `OPSPILOT_WORKER_POLL_INTERVAL` | `1s` | Worker task poll frequency |

## Architecture overview

This is a modular monolith with two entrypoints:

- **`cmd/api`** — HTTP server. Wires all services via constructor injection, serves REST + SSE + embedded admin pages.
- **`cmd/worker`** — Background task runner. Polls for queued workflow tasks and eval runs; also registers Temporal activities.
- **`cmd/ticketapi`** — Dev-only fake external ticket system for local compose stack.

### Core runtime flow

1. **Session / Chat** (`internal/session`, `internal/app/chat`) — in-memory session and message persistence, chat application service.
2. **Context Engine** (`internal/contextengine`) — layered context assembly (recent turns, scratchpad, profile, retrieval). Token-budgeted, not raw transcript.
3. **Planner** (`internal/agent/planner`) — typed, deterministic plan generation from structured context.
4. **Retrieval** (`internal/retrieval`) — structured query → retrieval with provenance (source ID, chunk ID, score, tenant scope).
5. **Tool Agent** (`internal/agent/tool`) — executes tools from registry. Side-effecting tools require approval gates.
6. **Critic** (`internal/agent/critic`) — validates answer quality, citations, policy risk.
7. **Workflow** (`internal/workflow`) — async task promotion (queued → running → succeeded/failed). `report_generation` tasks go through Temporal; `approved_tool_execution` tasks use approval-gated Temporal workflows.
8. **Report** (`internal/report`) — durable report read model from successful report_generation tasks.
9. **Cases** (`internal/case`) — operator case management linking follow-up work to tasks/reports.
10. **Eval** (`internal/eval`) — eval cases, datasets (draft→published), runs, judge runtime (placeholder or HTTP provider), aggregated eval reports with bad-case lineage.
11. **Versions** (`internal/version`) — durable runtime version registry for reproducibility.
12. **Trace Detail** (`internal/observability/tracedetail`) — lineage and provenance drill-down across tasks/reports/cases.

### Storage layer

- **`internal/storage/postgres`** — all pgx-based stores: workflow tasks, reports, versions, eval cases/datasets/runs/reports, cases. Each domain has a dedicated store file.
- **`internal/storage/migrate`** — migration plan execution from `db/migrations/`.
- In-memory stores exist for workflow, report, version (used in tests and as initial implementations).

### HTTP API layer

All handlers live in `internal/app/httpapi`. Key patterns:
- Middleware in `middleware.go` (request ID, tenant ID threading)
- Admin read models served from `internal/app/admin/taskboard`
- Embedded admin pages served from `web/admin` (task board, reports, cases, evals, datasets, runs, report compare, trace detail, version detail)

### Tool system

- `internal/tools/registry` — typed tool registry with read-only vs side-effecting classification
- `internal/tools/http/tickets` — HTTP ticket adapter with fake server for testing
- Tools are registered via `toolregistry.NewDefaultRegistryWithOptions`

### Key conventions

- Services use `NewServiceWithStore` / `NewServiceWithDependencies` constructors (manual DI).
- Domain packages expose `types.go` for models, `service.go` for logic, `doc.go` for package docs.
- Tests are colocated (`_test.go` in the same package).
- All env config flows through `internal/app/config/config.go` — no scattered `os.Getenv` calls.
- Prompts, rubrics, and eval fixtures are versioned as code under `eval/`.

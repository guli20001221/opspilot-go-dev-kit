# Local Bootstrap

## Scope

This runbook covers the current foundation slice only:

- Go module bootstrap
- API binary with `/healthz` and `/readyz`
- worker bootstrap
- local Docker Compose stack for PostgreSQL, Redis, Temporal, fake ticket API, API, and worker
- Make targets for format, test, build, and check

It does not yet wire real DB access from the app code or a real OpenTelemetry exporter.

## Prerequisites

- Go 1.24.2
- Optional: `make`
- PowerShell for the fallback script on Windows
- Docker Desktop with the daemon running

## Commands

1. Copy `.env.example` values into your local shell environment if you need overrides.
2. If `make` is installed, run `make test` and `make build`.
3. If `make` is not installed, run `powershell -File scripts/dev/tasks.ps1 test` and `powershell -File scripts/dev/tasks.ps1 build`.
4. Validate the Compose file with `docker compose config`.
5. Start the local stack with `make dev-up` or `powershell -File scripts/dev/tasks.ps1 dev-up`.
   This now runs `docker compose up -d --build`, so the app services start from prebuilt binaries rather than runtime `go run`.
6. Check `http://localhost:18080/healthz`.
7. Check `http://localhost:18080/readyz`.
8. Check Temporal UI at `http://localhost:8088`.
9. Check the fake ticket API at `http://localhost:19090/tickets/search?q=INC-100` with header `Authorization: Bearer local-dev-ticket-token` if you want to verify the HTTP adapter boundary directly.
10. Open `http://localhost:18080/admin/task-board` to inspect the embedded operator page against the local admin read model.

Successful build artifacts are emitted under `bin/`.

## Current API surface

- `POST /api/v1/sessions`
- `GET /api/v1/sessions/{session_id}/messages`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{task_id}`
- `POST /api/v1/tasks/{task_id}/approve`
- `POST /api/v1/tasks/{task_id}/retry`
- `GET /api/v1/admin/task-board`
- `GET /admin/task-board`
- `POST /api/v1/chat/stream`

The current chat stream implementation is a Milestone 1 skeleton:
- session storage is in-memory
- task storage is PostgreSQL-backed in the API runtime
- the worker process polls queued tasks and advances supported task types to terminal states
- `report_generation` is executed through a Temporal workflow on the `opspilot-report-tasks` queue when Temporal is enabled
- `approved_tool_execution` now starts a waiting Temporal workflow at task creation time and is resumed by the worker after the approval action updates the task row
- if `approved_tool_execution` fails after approval, the current Temporal run closes, the task row moves to `failed`, and `POST /api/v1/tasks/{task_id}/retry` starts a new failed-only Temporal run for the same task
- set `OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE=true` on the worker to force the first approval attempt to fail while keeping retry successful
- approval tasks promoted from chat now carry an internal tool payload so worker-side approved execution can run the registered tool after approval; manually created approval tasks without payload still use the compatibility path
- the local compose stack now starts a fake ticket API and routes the default ticket tools through `http://ticket-api:8090`
- set `OPSPILOT_TICKET_API_BASE_URL` yourself only when you want to override that default and target a different ticket service; outside compose, leaving it empty keeps the deterministic local ticket adapters
- approval-gated tasks can be resumed through the approval action endpoint
- failed tasks can be re-queued through the retry action endpoint
- task responses now include structured `audit_events`
- `GET /api/v1/tasks` now supports `tenant_id`, `status`, `task_type`, `reason`, `requires_approval`, `created_after`, `created_before`, `updated_after`, `updated_before`, `limit`, and `offset` filters for operator listing, with the time filters parsed as RFC3339 values, and returns `has_more` plus `next_offset` while keeping per-task `audit_events` only on `GET /api/v1/tasks/{task_id}`
- `GET /api/v1/admin/task-board` reuses the same filters but returns a backend task-board read model with visible-slice summary counts for the current page
- `GET /admin/task-board` is the first embedded operator UI and mirrors the same filters in a simple browser form while keeping all summary logic on the backend
- the same page now supports read-only task drill-down, so operators can inspect `audit_events`, `error_reason`, and `audit_ref` without leaving the board
- the detail panel also surfaces `Approve task` and `Retry task` controls when the current task state allows them, and those controls call the existing task action endpoints with the operator actor you enter on the page
- when a task has a Temporal-backed `audit_ref`, the same detail panel derives an `Open workflow history in Temporal UI` link so you can jump directly into the matching run
- enable `Auto refresh every 5s` on that same page when you want the board and selected task detail to keep tracking state changes without manual reload
- the local Compose app services now start from dedicated runtime images, which removes the previous startup dependence on downloading Go modules inside the running container
- the last successful `audit_event.detail` now carries an execution summary, such as which ticket comment was created
- failed `audit_event.detail` values now carry a coarse category prefix, such as `validation_error:` or `authorization_error:`
- failed tasks expose a summarized `error_reason` instead of the full wrapped Temporal error chain
- SSE always emits `meta`, `plan`, `state`, and `done`
- SSE may also emit `retrieval`, `tool`, and `task_promoted` depending on the internal runtime path
- assistant output is a fixed placeholder response
- the current HTTP contract is documented in `docs/openapi/openapi.yaml`

If your local PostgreSQL volume predates `db/migrations/000002_workflow_tasks.sql`, `db/migrations/000003_workflow_task_events.sql`, or `db/migrations/000004_workflow_task_payload.sql`, apply them manually before starting the API:

```powershell
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000002_workflow_tasks.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000003_workflow_task_events.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000004_workflow_task_payload.sql
```

If you change Compose environment variables such as `OPSPILOT_POSTGRES_DSN`, `OPSPILOT_TEMPORAL_ENABLED`, or `OPSPILOT_WORKER_POLL_INTERVAL`, recreate the app containers instead of only restarting them:

```powershell
docker compose up -d --build --force-recreate api worker
```

To override the built-in fake ticket API and point both app processes at a different ticket API, recreate them with:

```powershell
$env:OPSPILOT_TICKET_API_BASE_URL = "http://host.docker.internal:19090"
$env:OPSPILOT_TICKET_API_TOKEN = "secret-token"
docker compose up -d --build --force-recreate api worker
```

If an approval-gated task fails after approval, recover it with:

```powershell
$task = Invoke-RestMethod -Method Post -Uri http://localhost:18080/api/v1/tasks/<task_id>/retry -ContentType application/json -Body '{"actor":"operator-1"}'
Invoke-RestMethod -Uri "http://localhost:18080/api/v1/tasks/$($task.task_id)"
```

The expected progression is:
- task status changes from `failed` back to `queued`
- the worker claims it again
- the Temporal run referenced by `audit_ref` changes to a new run ID for the same `task_id`
- `audit_events` grows with `retried`, `claimed`, and the terminal action

To force this path locally without changing code, recreate only the worker with:

```powershell
$env:OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE = "true"
docker compose up -d --build --force-recreate worker
```

## Current gaps

- In the current Windows shell, `make` may be unavailable; use `scripts/dev/tasks.ps1` as the verified fallback.
- Redis is still present only as future infrastructure; no runtime code path uses it yet.
- The API process still exposes PostgreSQL task rows as the external task-status surface even when Temporal is driving report execution and approval waiting.
- No trace exporter exists yet; only request-scoped IDs are logged.

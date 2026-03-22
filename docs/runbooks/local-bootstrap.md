# Local Bootstrap

## Scope

This runbook covers the current foundation slice only:

- Go module bootstrap
- API binary with `/healthz` and `/readyz`
- worker bootstrap
- local Docker Compose stack for PostgreSQL, Redis, Temporal, API, and worker
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
6. Check `http://localhost:18080/healthz`.
7. Check `http://localhost:18080/readyz`.
8. Check Temporal UI at `http://localhost:8088`.

Successful build artifacts are emitted under `bin/`.

## Current API surface

- `POST /api/v1/sessions`
- `GET /api/v1/sessions/{session_id}/messages`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks/{task_id}`
- `POST /api/v1/tasks/{task_id}/approve`
- `POST /api/v1/tasks/{task_id}/retry`
- `POST /api/v1/chat/stream`

The current chat stream implementation is a Milestone 1 skeleton:
- session storage is in-memory
- task storage is PostgreSQL-backed in the API runtime
- the worker process polls queued tasks and advances supported task types to terminal states
- approval-gated tasks can be resumed through the approval action endpoint
- failed tasks can be re-queued through the retry action endpoint
- task responses now include structured `audit_events`
- SSE always emits `meta`, `plan`, `state`, and `done`
- SSE may also emit `retrieval`, `tool`, and `task_promoted` depending on the internal runtime path
- assistant output is a fixed placeholder response
- the current HTTP contract is documented in `docs/openapi/openapi.yaml`

If your local PostgreSQL volume predates `db/migrations/000002_workflow_tasks.sql`, apply it manually before starting the API:

```powershell
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000002_workflow_tasks.sql
```

If you change Compose environment variables such as `OPSPILOT_POSTGRES_DSN` or `OPSPILOT_WORKER_POLL_INTERVAL`, recreate the app containers instead of only restarting them:

```powershell
docker compose up -d --force-recreate api worker
```

## Current gaps

- In the current Windows shell, `make` may be unavailable; use `scripts/dev/tasks.ps1` as the verified fallback.
- The application opens PostgreSQL for workflow task persistence, but Redis and Temporal are not yet wired into runtime code paths.
- The API process opens PostgreSQL for workflow task persistence; the worker currently uses a placeholder poller rather than Temporal orchestration.
- No trace exporter exists yet; only request-scoped IDs are logged.

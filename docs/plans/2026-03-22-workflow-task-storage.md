# 2026-03-22 Workflow Task Storage

## Goal

Replace the API runtime's in-memory workflow task state with PostgreSQL-backed storage while keeping the public REST and SSE contracts stable.

## Scope

- add `workflow_tasks` migration
- add `OPSPILOT_POSTGRES_DSN` config
- add `pgxpool` connection wiring for the API process
- add PostgreSQL task repository under `internal/storage/postgres`
- make `internal/workflow.Service` store-driven
- keep unit tests on the in-memory store by default

## Non-goals

- Temporal SDK integration
- worker-driven task execution
- retry, cancel, or approval mutation endpoints

## Validation

- `go test ./internal/app/config -count=1`
- `go test ./internal/workflow -count=1`
- `go test ./internal/storage/postgres -count=1` with `OPSPILOT_TEST_POSTGRES_DSN`
- `go test ./...`
- `docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000002_workflow_tasks.sql`
- runtime verification of `POST /api/v1/tasks`, `GET /api/v1/tasks/{task_id}`, and `POST /api/v1/chat/stream`

# 2026-03-22 Worker Task Progression

## Goal

Advance stored workflow tasks beyond `queued` so the current async path is operationally visible instead of static.

## Scope

- add worker poll interval config
- add `running`, `succeeded`, and `failed` task states
- add task claim/update methods to the workflow store
- add placeholder worker execution for supported task types
- keep approval-gated tasks in `waiting_approval`

## Non-goals

- Temporal workflow execution
- approval resume APIs
- retry backoff or dead-letter handling

## Validation

- `go test ./internal/app/config -count=1`
- `go test ./internal/workflow -count=1`
- `go test ./internal/storage/postgres -count=1` with `OPSPILOT_TEST_POSTGRES_DSN`
- `go test ./internal/app/httpapi -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./...`
- `docker compose up -d --force-recreate api worker`
- create a task and verify it transitions out of `queued`

# 2026-03-22 Task Actions

## Goal

Expose the minimum approval and retry actions needed to move workflow tasks through the current placeholder async lifecycle.

## Scope

- add `POST /api/v1/tasks/{task_id}/approve`
- add `POST /api/v1/tasks/{task_id}/retry`
- reject invalid state transitions with explicit contract errors
- let approved tool-execution tasks complete through the placeholder worker path

## Non-goals

- full approval history tables
- multi-step approvals
- retry limits or backoff policies

## Validation

- `go test ./internal/workflow -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- `docker compose up -d --force-recreate api worker`
- runtime verification for approve and retry endpoints

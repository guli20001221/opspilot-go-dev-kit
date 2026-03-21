# 2026-03-22 Task Status API

## Goal

Expose the current in-memory workflow task records through a minimal REST contract so promoted tasks are externally visible and queryable.

## Scope

- share a single in-memory `workflow.Service` between chat orchestration and HTTP handlers
- add `POST /api/v1/tasks`
- add `GET /api/v1/tasks/{task_id}`
- preserve the existing SSE `task_promoted` event contract
- update OpenAPI, README, runbook, and API skill guidance

## Non-goals

- Temporal workflow execution
- durable task persistence
- retry or approval mutation endpoints

## Validation

- `go test ./internal/workflow -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- local host verification for task create and task lookup against the running API container

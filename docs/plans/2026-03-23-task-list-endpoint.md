# 2026-03-23 Task List Endpoint

## Goal

Add a minimal operator-facing `GET /api/v1/tasks` endpoint so the current workflow runtime exposes a summary list view in addition to single-task drill-down.

## Scope

- add `GET /api/v1/tasks`
- support `tenant_id`, `status`, `task_type`, `reason`, `requires_approval`, `created_after`, `created_before`, `updated_after`, `updated_before`, `limit`, and `offset` filters
- keep list payloads lightweight by omitting `audit_events`
- return `has_more` and `next_offset` so operators can page through results without switching to a heavy drill-down endpoint
- preserve newest-first ordering by `updated_at`
- cover both in-memory and PostgreSQL-backed workflow stores
- update OpenAPI, README, runbook, architecture notes, and API contract skills

## Key decisions

- no pagination cursor yet; keep the first slice simple with a bounded `limit`
- use the existing task response shape minus `audit_events` for the list surface
- fix task ID generation in `workflow.Service` so rapid in-memory task promotion does not overwrite prior entries during tests or local operator flows

## Validation

- targeted handler, workflow, and PostgreSQL store tests for task listing and unique task IDs
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- local compose smoke test for create, list, and filtered lookup paths

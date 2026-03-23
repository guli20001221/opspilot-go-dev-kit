# 2026-03-23 Admin Taskboard Detail Panel

## Goal

Turn the first embedded admin task board into a usable operator workflow by adding single-task drill-down without changing backend API contracts.

## Scope

- extend `web/admin/task-board.html` with an in-page task detail panel
- load detail from the existing `GET /api/v1/tasks/{task_id}` endpoint
- keep the page read-only
- preserve backend-owned summary and audit logic
- update docs and admin skill guidance

## Key decisions

- do not add a separate admin detail page yet; keep the operator in one surface
- continue to treat `/api/v1/admin/task-board` as the lightweight list view and `/api/v1/tasks/{task_id}` as the authoritative detail view
- sanitize browser-side string rendering so backend-provided task metadata is not injected into HTML directly

## Validation

- targeted page route tests
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke when a runnable local API is available

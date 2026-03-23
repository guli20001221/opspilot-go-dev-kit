# 2026-03-23 Admin Taskboard Detail Navigation

## Goal

Make the embedded task board faster for operator triage by letting the detail panel move through the currently visible slice and surface a compact execution digest at the top.

## Scope

- add `Previous visible` and `Next visible` controls to the embedded task detail panel
- keep navigation scoped to the current `/api/v1/admin/task-board` result set instead of inventing a second browser-side list model
- derive an execution summary and timeline digest from the existing single-task detail response
- keep using the existing `GET /api/v1/tasks/{task_id}` contract as the authoritative detail source
- update docs and admin skill guidance

## Key decisions

- do not add new backend fields or new admin endpoints for navigation
- keep visible-slice navigation strictly local to the current filtered board state
- derive digest cards from `audit_events`, `error_reason`, and `audit_ref` before considering backend aggregation

## Validation

- failing-then-passing page test for navigation and digest affordances
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke for `Previous visible` / `Next visible` and digest updates on the embedded page

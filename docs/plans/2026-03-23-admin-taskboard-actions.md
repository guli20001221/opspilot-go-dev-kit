# 2026-03-23 Admin Taskboard Actions

## Goal

Close the first operator task loop by letting the embedded task board execute existing task actions from the detail panel.

## Scope

- add `Approve task` and `Retry task` controls to the embedded `web/admin` task detail panel
- call the existing `POST /api/v1/tasks/{task_id}/approve` and `POST /api/v1/tasks/{task_id}/retry` endpoints
- keep operator actor explicit in the page
- refresh board and detail state after each action
- update docs and admin skill guidance

## Key decisions

- do not add new admin-only mutation APIs
- keep action visibility state-driven: approval only for `waiting_approval`, retry only for `failed`
- sanitize browser rendering and continue treating backend endpoints as the sole source of workflow truth

## Validation

- failing-then-passing page test for action affordances
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke for approve and retry flows on the embedded page

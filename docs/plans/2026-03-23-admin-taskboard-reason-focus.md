# 2026-03-23 Admin Task Board Reason Focus

## Goal
Add a reason-based triage shortcut to the embedded admin task board so operators can pivot from a selected task into the matching `reason` slice without manually re-entering filters.

## Scope
- extend the existing detail action row in `web/admin/task-board.html`
- reuse the current board filter model and URL state
- keep the backend contract unchanged
- update static page tests, docs, and the admin console skill guidance

## Design
- add a `Focus same reason` action beside the existing lane and status shortcuts
- when clicked, write the selected task's `tenant_id` and `reason` back into the existing filter form
- reset `offset` to `0` before reloading the board
- leave other filters untouched so the pivot stays lightweight and predictable

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke on `/admin/task-board` confirming the URL and visible slice update after clicking `Focus same reason`

# 2026-03-24 Admin Task Board Queue Focus

## Goal
Add a queue-oriented triage shortcut to the embedded admin task board so operators can pivot from a selected task into the matching operational queue without manually re-entering both state and approval filters.

## Scope
- extend the existing detail action row in `web/admin/task-board.html`
- reuse the current `status` and `requires_approval` filters and URL state
- keep the backend contract unchanged
- update static page tests, docs, and the admin console skill guidance

## Design
- add a `Focus same queue` action beside the existing lane, task-type, approval, reason, and status shortcuts
- when clicked, write the selected task's `tenant_id`, `status`, and `requires_approval` back into the existing filter form
- reset `offset` to `0` before reloading the board
- leave other filters untouched so the pivot remains predictable and composable with existing slices

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke on `/admin/task-board` confirming the URL and visible slice update after clicking `Focus same queue`

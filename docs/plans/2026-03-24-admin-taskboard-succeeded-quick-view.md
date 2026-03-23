# 2026-03-24 Admin Task Board Succeeded Quick View

## Goal
Add a terminal-success quick view to the embedded admin task board so operators can jump straight into the current succeeded slice without manually re-entering the `status` filter.

## Scope
- extend the existing quick-view strip in `web/admin/task-board.html`
- reuse the current `status` filter and URL state
- keep the backend contract unchanged
- update static page tests, docs, and the admin console skill guidance

## Design
- add a `Succeeded` button beside the existing quick views
- when clicked, clear the other quick-view fields and write `status=succeeded` back into the existing filter form
- reset `offset` to `0` before reloading the board
- keep the logic inside the existing `applyQuickView` function so all quick-view behavior remains centralized

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke on `/admin/task-board` confirming the URL and visible slice update after clicking `Succeeded`

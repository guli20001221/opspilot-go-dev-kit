# 2026-03-24 Admin Task Board Task Type Quick Views

## Goal
Add task-type quick views to the embedded admin task board so operators can jump straight into report or approved-tool slices without first selecting a task.

## Scope
- extend the existing quick-view strip in `web/admin/task-board.html`
- reuse the current `task_type` filter and URL state
- keep the backend contract unchanged
- update static page tests, docs, and the admin console skill guidance

## Design
- add `Report tasks` and `Approved tools` buttons beside the existing quick views
- when clicked, clear the other quick-view fields and write the target `task_type` back into the existing filter form
- reset `offset` to `0` before reloading the board
- keep the implementation inside the existing `applyQuickView` function so the quick-view model stays centralized

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke on `/admin/task-board` confirming the URL and visible slice update after clicking both new quick views

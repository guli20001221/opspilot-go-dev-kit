# 2026-03-24 Admin Taskboard Succeeded-Reports Quick View

## Goal

Add a small operator-facing quick-view preset for successful report tasks, while keeping the board contract-first and filter-driven.

## Scope

- add a `Succeeded reports` quick-view button to the embedded admin task board
- implement the preset by writing `status=succeeded` and `task_type=report_generation` back into the existing filter form
- keep routing on the existing `/api/v1/admin/task-board` endpoint
- update the static HTML assertion, operator docs, and admin-console skill guidance

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: from the task board base URL, click `Succeeded reports` and confirm the URL gains `status=succeeded&task_type=report_generation`, the controls update, and the board narrows to the successful report lane

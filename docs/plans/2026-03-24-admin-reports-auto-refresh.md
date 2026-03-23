# 2026-03-24 Admin Reports Auto Refresh

## Goal

Add lightweight report-lane monitoring so operators can watch newly completed reports arrive without manual reloads.

## Scope

- add an `Auto refresh every 5s` toggle to `/admin/reports`
- poll only the existing `/api/v1/admin/task-board` and `/api/v1/tasks/{task_id}` contracts
- guard against overlapping fetches in the browser
- update tests, docs, and admin-console skill guidance

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath|TestAdminReportsPageRendersHTML|TestAdminReportsPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: open `/admin/reports`, enable `Auto refresh every 5s`, and verify the page remains stable and continues to show the selected report detail while polling

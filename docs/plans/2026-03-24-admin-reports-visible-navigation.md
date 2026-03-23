# 2026-03-24 Admin Reports Visible Navigation

## Goal

Make the first report-focused admin page easier to triage by supporting adjacent navigation within the current visible report slice.

## Scope

- add `Previous visible` and `Next visible` controls to `/admin/reports`
- keep the selected report row visually synced with the detail pane
- avoid new backend contracts by navigating only within the current `/api/v1/admin/task-board` slice
- update tests, docs, and admin-console skill guidance

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath|TestAdminReportsPageRendersHTML|TestAdminReportsPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: open `/admin/reports`, verify the selected row is highlighted, and where multiple visible reports exist verify `Previous visible` / `Next visible` step the detail pane through that slice

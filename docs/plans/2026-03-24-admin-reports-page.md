# 2026-03-24 Admin Reports Page

## Goal

Add the first report-focused admin page without introducing a new backend contract.

## Scope

- add an embedded `/admin/reports` page
- derive the page entirely from the existing `/api/v1/admin/task-board` and `/api/v1/tasks/{task_id}` contracts
- keep the report lane fixed to `status=succeeded` and `task_type=report_generation`
- expose report task drill-down, task-board handoff, API detail, and Temporal history link when present
- add route coverage and page HTML assertions

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath|TestAdminReportsPageRendersHTML|TestAdminReportsPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: open `/admin/reports?tenant_id=<known report tenant>&limit=10` and verify it loads the successful report lane plus a report detail panel

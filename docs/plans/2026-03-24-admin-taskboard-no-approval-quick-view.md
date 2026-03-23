# 2026-03-24 Admin Taskboard No-Approval Quick View

## Goal

Add a small operator-facing quick-view preset for tasks that do not require human approval, while keeping the board contract-first and filter-driven.

## Scope

- add a `No approval` quick-view button to the embedded admin task board
- implement the preset by writing `requires_approval=false` back into the existing filter form
- keep routing on the existing `/api/v1/admin/task-board` endpoint
- update the static HTML assertion, operator docs, and admin-console skill guidance

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: from the task board base URL, click `No approval` and confirm the URL gains `requires_approval=false`, the select updates, and the visible slice narrows to the non-approval task lane

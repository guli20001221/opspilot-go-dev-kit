# 2026-03-24 Admin Taskboard Failed-Approvals Quick View

## Goal

Add a small operator-facing quick-view preset for failed approval-gated tasks, while keeping the board contract-first and filter-driven.

## Scope

- add a `Failed approvals` quick-view button to the embedded admin task board
- implement the preset by writing `status=failed` and `requires_approval=true` back into the existing filter form
- keep routing on the existing `/api/v1/admin/task-board` endpoint
- update the static HTML assertion, operator docs, and admin-console skill guidance

## Validation

- `go test ./internal/app/httpapi -run 'TestAdminTaskBoardPageRendersHTML|TestAdminTaskBoardPageRejectsUnknownSubpath' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke: from the task board base URL, click `Failed approvals` and confirm the URL gains `status=failed&requires_approval=true`, the controls update, and the board narrows to the failed approval lane

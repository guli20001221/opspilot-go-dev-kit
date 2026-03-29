# 2026-03-29 Eval Report List Linked Case Summary

## Goal
Expose canonical linked-case ownership and open-pressure on `GET /api/v1/eval-reports` so `/admin/eval-reports` list rows can show latest linked follow-up state without opening detail.

## Scope
- Add `linked_case_summary` to eval-report list rows.
- Render lightweight linked-open and latest-owner hints in the eval-report list UI.
- Keep detail and compare contracts unchanged.

## Validation
- Targeted eval-report HTTP and admin-page tests.
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

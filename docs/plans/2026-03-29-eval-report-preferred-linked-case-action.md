# 2026-03-29 Eval Report Preferred Linked Case Action

## Goal
Expose a canonical linked-case handoff decision on eval-report list/detail reads so `/admin/eval-reports` opens the latest linked case or the canonical queue without browser-side heuristics.

## Scope
- Add `preferred_linked_case_action` to eval-report list/detail contracts.
- Rewire `/admin/eval-reports` list and detail handoff links to consume it.
- Keep compare-specific handoff unchanged.

## Validation
- Targeted eval-report HTTP and admin-page tests.
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

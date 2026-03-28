# 2026-03-28 Eval Report Row Unresolved Bad-Case Handoff

## Goal
Let operators jump from an eval-report list row directly into that report's canonical unresolved bad-case slice.

## Why
- the list already exposes `bad_case_without_open_follow_up_count`
- the page already has a canonical unresolved slice via `bad_case_needs_follow_up=true`
- requiring a detail-pane round trip adds friction to triage

## Slice
1. add a row-level `Open unresolved bad cases` handoff on `/admin/eval-reports`
2. target `/admin/eval-reports?tenant_id=...&report_id=...&selected_report_id=...&bad_case_needs_follow_up=true`
3. keep `report_id` as the canonical list filter and use `selected_report_id` for the currently opened detail pane
4. keep the handoff hidden when `bad_case_without_open_follow_up_count == 0`
5. cover it in eval-report page runtime smoke

## Validation
- focused `go test` on admin eval-report page rendering and runtime smoke
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

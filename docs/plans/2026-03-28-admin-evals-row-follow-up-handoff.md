# 2026-03-28 admin evals row follow-up handoff

## Goal
Let operators jump from the visible `/admin/evals` queue directly into existing follow-up work without opening the detail pane first.

## Scope
- expose `Open latest case` on eval rows when `latest_follow_up_case_id` exists
- expose `Open queue` on eval rows using the canonical `source_eval_case_id` queue filter
- keep the existing detail handoff and backend contracts unchanged

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

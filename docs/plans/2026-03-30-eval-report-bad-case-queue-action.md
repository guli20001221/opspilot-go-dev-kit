# 2026-03-30 Eval Report Bad-Case Queue Action

## Goal
Move unresolved bad-case queue handoff for eval reports and compare sides into the canonical backend contract.

## Why
- `/admin/eval-reports` and `/admin/eval-report-compare` already had durable unresolved bad-case pressure.
- The queue handoff was still being reconstructed in browser code from `report_id`.
- That left operator routing logic duplicated outside the canonical read model.

## Slice
- add `preferred_bad_case_queue_action` to eval-report list/detail reads
- add the same field to compare-side reads
- switch admin pages to consume that field instead of building the unresolved-bad-case queue URL heuristically
- update OpenAPI, docs, and skill guidance

## Validation
- targeted `go test` for eval-report handlers and admin pages
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI parse validation

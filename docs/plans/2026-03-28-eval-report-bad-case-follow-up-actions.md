# 2026-03-28 Eval Report Bad-Case Follow-Up Actions

## Goal
Move bad-case follow-up handoff on `/admin/eval-reports` from browser-side heuristics to a backend-owned typed contract.

## Why
- per-bad-case rows already carry follow-up counts and latest case IDs
- the page was still recomputing `create` versus `open existing` in browser code
- eval-case, eval-report, and compare lanes already use backend-owned action fields

## Slice
1. add `preferred_follow_up_action` to each `bad_cases[]` item returned by `GET /api/v1/eval-reports/{report_id}`
2. derive that field from canonical eval-case follow-up summary
3. document it in OpenAPI
4. switch `/admin/eval-reports` bad-case rows to consume the typed field, keeping a compatibility fallback for older payloads

## Validation
- focused `go test` on eval-report contract and admin eval-reports page
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

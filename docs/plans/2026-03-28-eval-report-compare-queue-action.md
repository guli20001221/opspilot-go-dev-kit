# 2026-03-28 Eval Report Compare Queue Action

## Goal
Move compare-origin follow-up handoff on `/admin/eval-reports` behind one backend-owned action field.

## Why
- the canonical eval-report contract already carries compare follow-up summary
- the page was still deciding `open compare queue` by inspecting counts in browser code
- compare queue reuse should come from the same typed contract pattern as normal follow-up reuse

## Slice
1. add `preferred_compare_follow_up_action` to canonical eval-report list/detail reads
2. support `report_id` on `GET /api/v1/eval-reports` so row-level unresolved handoff can stay on one canonical list endpoint
3. update `/admin/eval-reports` list/detail to consume the typed compare-queue action
4. remove count-based compare queue heuristics from the page

## Validation
- focused `go test` on eval-report HTTP contract and admin page smoke
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

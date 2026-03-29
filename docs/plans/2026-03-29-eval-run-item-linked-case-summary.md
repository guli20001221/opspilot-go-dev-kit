## Goal

Remove the last obvious browser-side gap in `/admin/eval-runs` item triage by exposing per-item linked-case pressure from the canonical run detail contract.

## Why

`latest_follow_up_case_id` alone is not enough for operators to judge whether a failed eval item has zero, one, or many linked follow-up cases, or whether the latest one is still open. That decision should come from the backend-owned read model, not from page heuristics.

## Scope

- add `linked_case_summary` to `GET /api/v1/eval-runs/{run_id}` on both `items[]` and `item_results[]`
- keep `GET /api/v1/eval-runs` list contract unchanged
- render the new summary in `/admin/eval-runs`
- update OpenAPI, docs, and skill guidance

## Validation

- targeted `go test` for `internal/app/httpapi`
- `go test ./...`
- OpenAPI parse check
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

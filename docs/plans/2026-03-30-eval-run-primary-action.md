# 2026-03-30 Eval Run Primary Action

## Goal

Remove the remaining browser-side primary-action routing from `/admin/eval-runs` by exposing a canonical `preferred_primary_action` on eval-run `items[]` and `item_results[]`.

## Scope

- add `preferred_primary_action` to `GET /api/v1/eval-runs/{run_id}` detail rows
- update `/admin/eval-runs` to consume the backend-owned field for the main per-result action
- keep `preferred_linked_case_action` as the secondary linked-case handoff
- update OpenAPI and operator docs

## Non-goals

- no new endpoints
- no changes to case deduplication semantics
- no changes to eval-run list filtering

## Validation

- focused `go test` on `internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

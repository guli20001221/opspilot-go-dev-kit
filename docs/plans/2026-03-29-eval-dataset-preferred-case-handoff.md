# 2026-03-29 Eval Dataset Preferred Case Handoff

## Goal
Remove browser-side priority logic from `/admin/eval-datasets` when deciding whether the main case handoff should open the dataset-wide follow-up queue or the latest-report follow-up queue.

## Change
- add `preferred_case_handoff_action` to canonical `GET /api/v1/eval-datasets` and `GET /api/v1/eval-datasets/{dataset_id}`
- define it as a backend-owned choice that prefers dataset-wide case reuse when available, otherwise falls back to the latest-report follow-up queue action
- update `/admin/eval-datasets` to consume that single action instead of composing queue priority in page helpers

## Why
- dataset queue priority is operator policy, not frontend state
- the page already had two lower-level queue actions; the main handoff was still stitched together in JavaScript
- this keeps the dataset lane aligned with the same backend-owned action pattern already used by evals, eval reports, compare, and eval runs

## Validation
- targeted `go test` for eval dataset API and admin page coverage
- `go test ./...`
- OpenAPI parse check
- repo `check` task

# 2026-03-30 Eval Dataset Item Linked Case Actions

## Goal
Expose member-level linked-case pressure and handoff on canonical eval-dataset detail reads.

## Why
- `/admin/eval-datasets` already had durable dataset, recent-run, and latest-report follow-up actions.
- Individual dataset members still only exposed raw eval-case IDs, so member-level linked-case handoff was left to browser logic.
- This slice keeps operator routing backend-owned and aligned with the rest of the eval lanes.

## Slice
- add `linked_case_summary` to `GET /api/v1/eval-datasets/{dataset_id}` `items[]`
- add `preferred_linked_case_action` to the same `items[]`
- wire `/admin/eval-datasets` dataset-member handoff from those fields
- update OpenAPI, docs, and skill guidance

## Validation
- targeted `go test` for eval-dataset handlers and admin page HTML
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI parse validation

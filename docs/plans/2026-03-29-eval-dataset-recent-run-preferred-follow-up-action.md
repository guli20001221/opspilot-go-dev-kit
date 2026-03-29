## Goal

Push recent eval activity on `/admin/eval-datasets` one step further toward backend-owned operator routing.

## Scope

- extend `GET /api/v1/eval-datasets/{dataset_id}` so each `recent_runs[]` row carries a typed `preferred_follow_up_action`
- keep action modes aligned with the existing dataset-level follow-up handoff:
  - `none`
  - `open_latest_report_queue`
  - `open_latest_run_queue`
- update `/admin/eval-datasets` to consume that row-level action instead of inferring report-vs-run queue routing from `report_id` and `needs_follow_up`

## Why

Before this slice, dataset detail recent activity still left one routing decision in the browser:

- if one recent run had unresolved failed items and a durable report, open the report queue
- otherwise, fall back to the run queue

That decision is now part of the canonical read model, which keeps operator handoff stable across consumers and avoids duplicating queue logic in the page.

## Validation

- targeted `go test` for dataset detail and admin page smoke
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI YAML parse check

# 2026-03-29 Eval Dataset Run-Backed Case Summary

## Goal
Expose latest-run follow-up ownership directly on canonical eval-dataset reads so operators can see whether the newest durable run already has claimed work without leaving `/admin/eval-datasets`.

## Change
- add `run_backed_case_summary` to `GET /api/v1/eval-datasets` and `GET /api/v1/eval-datasets/{dataset_id}`
- derive it from durable `source_eval_run_id` case lineage for the latest run only
- surface the summary and a direct `Open latest run-backed case` handoff on `/admin/eval-datasets`

## Why
- dataset pages already show latest run/report pressure, dataset-wide case pressure, and latest-report case pressure
- run-backed follow-up was still only visible after a second hop into eval-run or case lanes
- this keeps latest-run ownership on the same backend-owned read model as the rest of dataset triage

## Validation
- targeted eval-dataset API tests
- admin page runtime smoke for `/admin/eval-datasets`
- `go test ./...`
- OpenAPI parse check
- repo `check` task

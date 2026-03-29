# 2026-03-29 Case Eval Run Queue

## Goal

Extend the canonical case contract so follow-up opened from the eval-run lane preserves durable `source_eval_run_id` lineage and can be triaged as a first-class `/admin/cases` queue.

## Scope

- add `source_eval_run_id` to durable case storage and HTTP contract
- support `source_eval_run_id` and `run_backed_only` on `GET /api/v1/cases`
- preserve run lineage when `/admin/eval-runs` creates a follow-up case
- expose `Run-backed cases` plus eval-run handoff on `/admin/cases`

## Validation

- targeted HTTP and Postgres store tests for create/get/list
- admin cases HTML and runtime smoke coverage for run-backed case handoff
- OpenAPI parse validation
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

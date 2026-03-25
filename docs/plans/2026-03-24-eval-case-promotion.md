# 2026-03-24 eval case promotion

## Goal

Land the smallest durable regression-promotion slice from `/admin/cases` without inventing a second admin-only write path.

## Scope

- add durable `eval_cases` storage
- add `internal/eval` service and PostgreSQL store
- add `POST /api/v1/eval-cases`
- add `GET /api/v1/eval-cases/{eval_case_id}`
- add `/admin/cases` action `Promote to eval`

## Contract decisions

- eval promotion is rooted in `source_case_id`
- create is idempotent by `source_case_id`
- lineage is copied from canonical case plus trace drill-down:
  - `source_task_id`
  - `source_report_id`
  - `trace_id`
  - `version_id`
- UI only calls canonical REST endpoints and deep-links to the returned eval-case API detail

## Verification

- targeted Go tests for `internal/eval`, `internal/storage/postgres`, and `internal/app/httpapi`
- `go test ./...`
- OpenAPI YAML parse
- local migration apply for `000012_eval_cases.sql`
- REST smoke for create/get eval case
- browser smoke for `/admin/cases` promote action

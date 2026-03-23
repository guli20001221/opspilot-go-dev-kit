# 2026-03-24 Durable Report Read Model

## Goal
Add the first durable report contract so successful `report_generation` tasks produce a report entity instead of only task audit history.

## Scope
- add an additive `reports` migration
- persist report rows from the worker-side success path for `report_generation`
- add a minimal `internal/report` service and PostgreSQL store
- expose `GET /api/v1/reports/{report_id}`
- update OpenAPI, README, architecture, runbook, and relevant skills

## Non-goals
- no report list endpoint
- no case contract
- no new admin page in this slice
- no separate report comparison workflow yet

## Design
- treat task status and report artifact as separate read models
- derive a stable `report_id` as `report-<task_id>`
- keep report persistence in worker-side orchestration code, not inside Temporal workflow definitions
- store basic report metadata plus a JSON metadata blob carrying task identity and execution summary

## Validation
- targeted Go tests for `internal/report`, `internal/storage/postgres`, `internal/workflow`, and `internal/app/httpapi`
- `go test ./...`
- repo `check` target
- local smoke: create a `report_generation` task, wait for success, then query `GET /api/v1/reports/report-<task_id>`

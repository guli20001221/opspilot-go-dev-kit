# 2026-03-28 Admin Cases Eval-Backed Slice

## Goal

Turn durable `source_eval_report_id` lineage into a real operator slice instead of leaving it as detail-only metadata.

## Scope

- add `source_eval_report_id` filtering to `GET /api/v1/cases`
- add `eval_backed_only=true` filtering to `GET /api/v1/cases`
- wire `/admin/cases` to those canonical filters
- expose an `Eval-backed cases` quick view in `/admin/cases`
- sync OpenAPI, runbook, README, architecture notes, and skills

## Non-goals

- no new case entity types
- no eval-only backlog or duplicate queue
- no new admin-only API
- no case detail contract changes beyond existing `source_eval_report_id`

## Validation

- targeted `go test` for `internal/app/httpapi`
- targeted PostgreSQL filter test for `internal/storage/postgres`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-28 Eval Report Follow-up Summary

## Goal

Surface operator-facing follow-up case pressure directly on the canonical eval-report list contract and reuse it in `/admin/eval-reports`.

## Scope

- add durable follow-up case summary aggregation by `source_eval_report_id`
- expose summary fields on `GET /api/v1/eval-reports`
- render the new summary on `/admin/eval-reports`
- add memory-store and PostgreSQL regression coverage
- sync OpenAPI, docs, and admin/eval skills

## Non-goals

- no new eval-report detail fields beyond the list summary
- no new case-specific API for eval reports
- no new admin-only read model

## Validation

- targeted `go test` for `internal/case`, `internal/storage/postgres`, and `internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI YAML parse check

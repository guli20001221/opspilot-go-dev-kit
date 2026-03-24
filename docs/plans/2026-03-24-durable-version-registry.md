# 2026-03-24 Durable Version Registry

## Goal

Land a durable runtime version registry so operators can explain which runtime bundle produced a task, report, or trace drill-down without reconstructing that metadata from ad hoc task payloads.

## Scope

- add additive PostgreSQL migrations for `versions` plus `version_id` references on tasks and reports
- add `internal/version` service and PostgreSQL store
- expose `GET /api/v1/versions` and `GET /api/v1/versions/{version_id}`
- thread `version_id` through task, report, and trace-drilldown contracts
- add `/admin/version-detail` and handoffs from report, report-compare, and trace pages
- update OpenAPI, runbook, architecture notes, and relevant skills

## Non-goals

- provider-specific runtime capture beyond the current skeleton defaults
- version mutation endpoints
- a separate admin-only version API

## Validation

- targeted Go tests for `internal/version`, PostgreSQL stores, report integration, trace drill-down, and HTTP handlers
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- Docker smoke for migrations, `/api/v1/versions`, `/api/v1/versions/{version_id}`, `/admin/version-detail`, and version handoffs from report/report-compare/trace pages

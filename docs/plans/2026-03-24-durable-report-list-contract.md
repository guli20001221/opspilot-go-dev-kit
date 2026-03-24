# 2026-03-24 durable report list contract

## Goal

Promote reports from a detail-only durable artifact into a canonical listable object with its own REST read surface.

## Why this slice

The repository already had:

- durable `GET /api/v1/reports/{report_id}`
- a durable case contract
- `/admin/reports`

But the reports page still derived its table from task-board slices. That kept operator report browsing coupled to workflow task state instead of the report artifact itself.

## Delivered

- `GET /api/v1/reports` with:
  - `tenant_id`
  - `status`
  - `report_type`
  - `source_task_id`
  - `limit`
  - `offset`
- durable list support in:
  - `internal/report`
  - `internal/storage/postgres`
  - memory-backed report store
- `/admin/reports` now sources its list from the durable report endpoint
- `/admin/reports` still uses task detail only for:
  - audit timeline
  - execution summary
  - Temporal deep links
  - case handoff provenance

## Design notes

- Report ordering is newest-ready first, with deterministic tie-breaks.
- The list contract stays artifact-focused; it does not embed workflow audit history.
- The page remains contract-first by combining:
  - durable report list
  - durable report detail
  - canonical task detail for provenance

## Validation

- targeted `go test` for `internal/report`, `internal/storage/postgres`, `internal/app/httpapi`
- `go test ./...`
- OpenAPI YAML parse validation
- browser smoke for `/admin/reports`

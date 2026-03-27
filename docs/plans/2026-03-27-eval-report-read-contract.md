# 2026-03-27 Eval Report Read Contract

## Goal

Expose the durable aggregated eval-report artifact through canonical REST reads before any eval-report-heavy operator UI lands.

## Scope

- add tenant-scoped `GET /api/v1/eval-reports`
- add tenant-scoped `GET /api/v1/eval-reports/{report_id}`
- keep the list response lightweight
- reserve `metadata` and `bad_cases` for single-report detail reads
- wire the API runtime to the existing durable eval-report store

## Notes

- the durable report artifact already exists in `internal/eval`
- this slice does not add an admin page yet
- this slice keeps report aggregation logic in the backend and only exposes typed read contracts

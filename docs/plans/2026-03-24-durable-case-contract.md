# 2026-03-24 durable case contract

## Goal

Land the first durable operator case object so follow-up work is no longer stranded across task and report views.

## Scope

- add a `cases` table with stable IDs and source links
- add `internal/case` service and memory-backed default store
- add a PostgreSQL store for durable case records
- add `POST /api/v1/cases`
- add `GET /api/v1/cases/{case_id}`
- validate tenant-safe source linkage against existing task and report contracts

## Notes

- this slice deliberately does not add case list views or `/admin/cases`
- cases are currently operator-managed records, not workflow-managed state machines
- source lineage is stable through `source_task_id` and `source_report_id`

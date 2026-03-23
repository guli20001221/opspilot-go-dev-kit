# 2026-03-24 Admin Reports Read From Report Endpoint

## Goal
Move `/admin/reports` detail off task-only semantics so the page consumes the stable report contract for artifact metadata.

## Scope
- keep the existing report lane list based on `/api/v1/admin/task-board`
- fetch `GET /api/v1/reports/{report_id}` for selected report metadata
- keep `GET /api/v1/tasks/{task_id}` for audit timeline and Temporal provenance
- update page handoff links and HTML route assertions

## Non-goals
- no new report list endpoint
- no case view
- no change to `/admin/task-board`

## Design
- derive `report_id` as `report-<task_id>` from the current successful report task
- render title, summary, report status, and ready time from the durable report response
- keep audit timeline and Temporal link on the existing task response
- preserve current shareable URL and visible-slice navigation model

## Validation
- admin HTML route tests
- `go test ./...`
- repo `check` target
- browser smoke on `/admin/reports`

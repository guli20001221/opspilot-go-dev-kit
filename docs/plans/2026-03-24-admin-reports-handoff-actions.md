# 2026-03-24 Admin Reports Handoff Actions

## Goal
Add operator-facing handoff actions to `/admin/reports` without introducing any new backend contract.

## Scope
- add `Copy report summary` to the report detail panel
- add `Copy report link` to the report detail panel
- keep the implementation derived entirely from the existing `/api/v1/tasks/{task_id}` response
- keep `/admin/reports` fixed to the current successful report lane

## Non-goals
- no new report-specific REST endpoint
- no export/download API
- no separate report domain model

## Design
- reuse the selected report task detail as the single source of truth
- derive a compact clipboard summary from `task_id`, status, tenant, session, audit ref, error reason, and `audit_events`
- copy the current page URL for shareable filtered report-lane handoff
- surface copy success or failure through the existing report detail status note

## Validation
- HTML route tests for `/admin/reports`
- `go test ./...`
- repo `check` target
- browser smoke for copy-action state changes on the embedded reports page

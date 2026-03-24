# 2026-03-24 Trace Drill-Down Minimum Slice

## Goal

Land the smallest operator-facing trace drill-down slice:

- a read-only trace lookup contract over one durable task, report, or case
- a minimal `/admin/trace-detail` page
- handoff links from the existing task, report, compare, and case pages

## Why now

The repository now has durable task, report, and case contracts plus operator pages for each lane.
The next horizontal gap is traceability: operators can see object state, but they still lack one canonical page that answers "which lineage produced this object, and where is the current Temporal or request-level provenance?"

## Scope

### Backend

- add `internal/observability/tracedetail`
- add `GET /api/v1/trace-drilldown`
- require exactly one of `task_id`, `report_id`, or `case_id`
- return durable lineage plus current request/session/audit/Temporal pointers

### Frontend

- add `/admin/trace-detail`
- add task/report/case/compare handoff links into the new page
- keep all trace rendering derived from the backend contract

### Docs

- update OpenAPI
- update README and architecture notes
- sync admin/API/observability skills

## Non-goals

- no real trace backend search UI
- no Langfuse or vendor-specific trace explorer
- no span-level timeline reconstruction
- no live subscriptions

## Validation

- targeted `go test` for `internal/observability/tracedetail` and `internal/app/httpapi`
- `go test ./...`
- OpenAPI YAML parse check
- browser smoke on `/admin/trace-detail`

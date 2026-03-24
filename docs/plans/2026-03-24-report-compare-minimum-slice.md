# 2026-03-24 Report Compare Minimum Slice

## Goal

Land the smallest durable report-comparison vertical slice:

- a read-only compare contract over two durable report IDs
- a minimal `/admin/report-compare` operator page
- report-lane handoff into the comparison page

## Why now

The repository already has durable task, report, and case contracts plus operator pages for each lane.
The next highest-leverage operator gap is no longer "can we inspect one object" but "can we compare two durable report artifacts without recomputing the diff in the browser."

## Scope

### Backend

- add `report.Comparison` / `ComparisonSummary`
- add `report.Service.CompareReports`
- add `GET /api/v1/report-compare?left_report_id=...&right_report_id=...`
- keep the contract read-only and typed

### Frontend

- add `/admin/report-compare`
- accept `left_report_id` and `right_report_id`
- show a narrow comparison summary plus left/right report drill-down
- reuse existing report API links for handoff

### Docs

- update OpenAPI
- update README and architecture notes
- sync admin/API skill guidance

## Non-goals

- no report version graph
- no admin-only compare mutation surface
- no workflow-task diffing inside the compare contract
- no case integration in this slice

## Validation

- targeted `go test` for `internal/report` and `internal/app/httpapi`
- `go test ./...`
- OpenAPI YAML parse check
- browser smoke against two real durable report rows

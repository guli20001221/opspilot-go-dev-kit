# 2026-03-24 Report Case Handoff Correctness

## Goal

Close two operator-facing correctness gaps in the report-to-case handoff paths:

- creating a case from a successful report task in `/admin/task-board` must preserve `source_report_id`
- `/admin/reports` must not offer `Create case` while the page is in task-only fallback mode without a durable report row

## Scope

- add deterministic report ID handoff on the task-board create-case path
- degrade task-board case creation to task-only handoff when durable report lookup is missing or temporarily unavailable
- disable report-originated case creation when `currentReportDetail` is missing
- add lightweight regression assertions in the admin HTML tests
- verify both flows with browser smoke

## Validation

- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- browser smoke for task-board report case creation
- browser smoke for reports fallback mode

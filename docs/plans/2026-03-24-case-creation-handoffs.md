# 2026-03-24 Case Creation Handoffs

## Scope
- add `Create case` to `/admin/task-board` detail
- add `Create case` to `/admin/reports` detail
- reuse existing `POST /api/v1/cases`
- deep-link success to `/admin/cases?tenant_id=...&case_id=...`

## Notes
- task board creates task-sourced cases from the currently selected task
- reports page creates report-sourced cases when a durable report row exists, and falls back to task-sourced case creation if the report row is missing
- failures such as `409 invalid_case_source` are surfaced inline in the existing detail status note

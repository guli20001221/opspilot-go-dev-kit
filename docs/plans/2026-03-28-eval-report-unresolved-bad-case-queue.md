# 2026-03-28 Eval Report Unresolved Bad-Case Queue

## Goal
Lift unresolved bad-case follow-up pressure onto the canonical eval-report list contract so operators can browse the queue without opening each report detail first.

## Scope
- add durable `bad_case_without_open_follow_up_count` to `GET /api/v1/eval-reports`
- add canonical list filter `bad_case_needs_follow_up=true|false`
- wire `/admin/eval-reports` to that list filter through one quick view and one lightweight row summary
- keep bad-case drill-down filtering on `GET /api/v1/eval-reports/{report_id}` as a separate detail concern

## Notes
- report-level `needs_follow_up` remains about open follow-up cases linked to the report as a whole
- `bad_case_needs_follow_up` is a separate signal: at least one bad case lacks an open linked follow-up case
- the admin page keeps these as separate filters so operator queues stay explicit

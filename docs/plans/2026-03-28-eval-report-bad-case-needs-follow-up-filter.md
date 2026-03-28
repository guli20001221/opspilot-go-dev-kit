# 2026-03-28 Eval Report Bad-Case Needs Follow-Up Filter

## Goal
Add a backend-owned `bad_case_needs_follow_up` filter to the canonical eval-report detail contract so `/admin/eval-reports` can isolate unresolved bad cases without browser-only filtering.

## Scope
- accept `bad_case_needs_follow_up=true|false` on `GET /api/v1/eval-reports/{report_id}`
- filter only the heavy `bad_cases` drill-down, without changing top-line report metrics
- add detail-level quick views in `/admin/eval-reports`
- update OpenAPI, docs, and skill guidance

## Non-goals
- no changes to `GET /api/v1/eval-reports` list semantics
- no new eval-only case queue
- no new backend endpoint

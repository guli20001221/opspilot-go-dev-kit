# 2026-03-28 Eval-Report Preferred Follow-Up Action

## Goal

Move the report-level create-versus-reuse follow-up decision for `/admin/eval-reports` out of browser heuristics and into the canonical eval-report detail contract.

## Scope

- add typed `preferred_follow_up_action` to `GET /api/v1/eval-reports/{report_id}`
- keep the field backend-owned and derived from canonical eval-report follow-up summary
- switch only the report-level primary action on `/admin/eval-reports` to consume the field
- keep bad-case row actions unchanged in this slice
- update OpenAPI, README, architecture docs, runbook, and admin skill guidance

## Notes

- this slice does not add a new endpoint
- this slice does not change case deduplication behavior
- the page keeps a small fallback heuristic only for compatibility with older payloads during rollout

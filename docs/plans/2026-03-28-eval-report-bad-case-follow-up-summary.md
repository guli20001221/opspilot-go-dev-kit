# 2026-03-28 Eval Report Bad-Case Follow-Up Summary

## Goal
Expose per-bad-case follow-up case summary on the canonical eval-report detail contract so `/admin/eval-reports` can hand operators from one failing eval case straight into its freshest durable follow-up or the full bad-case case slice.

## Scope
- extend `GET /api/v1/eval-reports/{report_id}` bad-case payload with follow-up counts and latest follow-up case metadata
- reuse canonical case-summary reads keyed by `source_eval_case_id`
- surface `Open latest bad-case case` and `Open bad-case follow-up slice` in `/admin/eval-reports`
- update OpenAPI, docs, and matching skill guidance

## Non-goals
- no new eval-report list fields
- no new case endpoints
- no admin-only follow-up APIs

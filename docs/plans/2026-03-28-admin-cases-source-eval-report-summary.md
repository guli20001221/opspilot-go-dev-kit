# 2026-03-28 Admin Cases Source Eval Report Summary

## Goal
Keep `/admin/cases` contract-first while making eval-backed cases readable without forcing operators to jump out to `/admin/eval-reports` immediately.

## Slice
- reuse canonical `GET /api/v1/eval-reports/{report_id}` when a case has `source_eval_report_id`
- render a small source-eval-report summary card in the case detail pane
- keep existing handoff links to `/admin/eval-reports` and `/api/v1/eval-reports/{report_id}`
- degrade gracefully if the eval report row is missing

## Notes
- no backend contract changes
- no case payload changes
- smoke-test both the success path and the missing-report fallback in the browser path when Playwright is available

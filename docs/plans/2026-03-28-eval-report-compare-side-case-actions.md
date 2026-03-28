# 2026-03-28 Eval Report Compare Side Case Actions

## Goal
Let operators create a durable follow-up case from either side of an eval-report comparison without assuming the right-hand report is always the regression source.

## Scope
- replace the single compare-page `Create case` action with `Create case from left` and `Create case from right`
- preserve the canonical `POST /api/v1/cases` write path
- preserve deep-link handoff into `/admin/cases`
- add browser-level regression coverage proving left and right actions persist the expected `source_eval_report_id`

## Notes
- no new backend contract is introduced in this slice
- compare remains a read-only canonical diff surface; only case creation reuses the existing durable case API

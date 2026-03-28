# 2026-03-28 Eval Report Compare Linked Case Handoff

## Goal
Let operators jump from either side of an eval-report comparison into the full canonical case slice already linked to that side's `source_eval_report_id`.

## Scope
- add `Open left linked cases` and `Open right linked cases` to `/admin/eval-report-compare`
- reuse `/admin/cases?tenant_id=...&source_eval_report_id=...`
- add runtime coverage proving each side points to the expected canonical cases filter

## Notes
- no backend changes are needed because the canonical case list already supports `source_eval_report_id`

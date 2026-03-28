# 2026-03-28 Eval Report Compare Unresolved Bad-Case Handoff

## Goal
Expose unresolved bad-case pressure directly on each side of the canonical eval-report compare contract, then hand operators into the existing unresolved report lane instead of inventing compare-only queue state.

## Scope
- add per-side `bad_case_without_open_follow_up_count` to `GET /api/v1/eval-report-compare`
- show that count in `/admin/eval-report-compare`
- add side-specific handoff links into `/admin/eval-reports?bad_case_needs_follow_up=true&report_id=...`

## Notes
- compare stays read-only
- unresolved bad-case queue ownership remains with the canonical eval-report lane

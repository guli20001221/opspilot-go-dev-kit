# 2026-03-30 Eval Dataset Recent Run Primary Actions

## Goal
Move the main `/admin/eval-datasets` recent-run handoff from browser-owned routing into the canonical dataset detail contract.

## Scope
- add `preferred_primary_action` to `recent_runs[]` on `GET /api/v1/eval-datasets/{dataset_id}`
- keep existing `preferred_follow_up_action` and `preferred_case_action` as secondary handoffs
- switch `/admin/eval-datasets` recent activity to consume the new backend-owned primary action

## Notes
- priority order is:
  1. reuse an open run-backed case
  2. open the canonical unresolved queue
  3. open the durable eval report
  4. open the eval run
- this is additive and does not change existing queue/case semantics

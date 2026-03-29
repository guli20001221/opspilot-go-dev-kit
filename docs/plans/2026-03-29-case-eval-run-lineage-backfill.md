# 2026-03-29 Case Eval-Run Lineage Backfill

## Goal
Backfill `cases.source_eval_run_id` for older eval-run follow-up rows so canonical run-backed queues and summaries stay correct for pre-lineage data.

## Strategy
- keep the migration additive and idempotent
- only backfill rows that match the old eval-run handoff summary template:
  `Follow up eval run <run_id> result for <eval_case_id>`
- additionally require matching `eval_run_items(run_id, eval_case_id)` membership and tenant alignment

## Why this shape
- older `/admin/eval-runs` follow-up writes encoded run lineage in summary text
- `source_eval_case_id` alone is not enough because the same eval case can appear in multiple runs
- ambiguous rows should remain untouched instead of being guessed onto the wrong run

## Validation
- extend test migration bootstrap to include the new SQL file
- add a PostgreSQL regression test that inserts a legacy row with no `source_eval_run_id`, reruns the migration, and verifies the backfilled lineage

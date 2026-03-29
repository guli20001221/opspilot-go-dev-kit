# 2026-03-29 Case Eval Run Dedupe

## Goal
Make run-backed follow-up creation idempotent at the canonical case contract level.

## Slice
- add typed `FindOpenCaseBySourceEvalRun` lookup in `internal/case`
- reuse the newest open case for the same `tenant_id + source_eval_run_id` inside `POST /api/v1/cases`
- keep `/admin/eval-runs` on the existing canonical write path instead of introducing page-only dedupe
- cover the reuse behavior with HTTP tests

## Notes
- this keeps one open queue item per eval run unless an operator explicitly closes or otherwise changes the workflow
- the behavior is additive and preserves existing run-backed queue reads via `source_eval_run_id` and `run_backed_only`

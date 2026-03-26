# 2026-03-27 Eval Run Retry

## Goal

Add the smallest operator recovery slice for durable eval runs by allowing a failed run to be re-queued on the same canonical record.

## Scope

- add `RetryRun` to `internal/eval.RunService`
- expose `POST /api/v1/eval-runs/{run_id}/retry`
- show `Retry run` in `/admin/eval-runs` only for failed runs
- keep retry on the same durable run row
- clear `error_reason`, `started_at`, and `finished_at` on retry

## Non-goals

- no judge prompts
- no per-item eval scoring
- no new run-attempt/history model
- no Temporal orchestration

## Validation

- service tests for retry state transitions
- worker-flow test proving retried runs are claimable again
- HTTP tests for success and invalid-state retry
- PostgreSQL integration test proving cleared timestamps and re-claimability

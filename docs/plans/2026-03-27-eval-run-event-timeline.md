# 2026-03-27 Eval Run Event Timeline

## Goal

Preserve durable eval-run lifecycle history even when retry clears the top-level failure fields on the canonical run record.

## Scope

- add append-only `eval_run_events`
- write `created`, `claimed`, `failed`, `retried`, and `succeeded` together with matching run state changes
- expose the timeline on `GET /api/v1/eval-runs/{run_id}`
- render the timeline on `/admin/eval-runs`

## Non-goals

- no per-item scoring
- no run-attempt object model
- no Temporal orchestration
- no heavy timeline data on `GET /api/v1/eval-runs`

## Validation

- service timeline ordering tests
- PostgreSQL timeline persistence tests
- HTTP detail response tests
- admin page static contract tests

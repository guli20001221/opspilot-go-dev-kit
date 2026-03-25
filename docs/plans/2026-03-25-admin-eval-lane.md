# 2026-03-25 Admin Eval Lane

## Goal

Turn durable eval-case promotion into a browsable operator lane by adding:

- `GET /api/v1/eval-cases`
- `/admin/evals`

## Why now

The repository already has durable tasks, cases, reports, versions, and single-record eval-case promotion. Without an eval queue, promoted coverage remains a write-only side path reachable only from one case at a time.

## Scope

- add tenant-scoped eval-case list filtering and pagination
- keep eval detail on the canonical `GET /api/v1/eval-cases/{eval_case_id}` contract
- add a read-only admin lane with handoff links into case, task, report, trace, and version surfaces

## Non-goals

- dataset mutation
- regression-run creation
- judge execution
- admin-only eval write paths

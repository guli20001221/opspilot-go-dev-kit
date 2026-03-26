# Eval Run Item Snapshot

## Goal

Make a durable eval run self-contained by snapshotting the published dataset membership onto the run at kickoff time.

## Scope

- add additive PostgreSQL storage for `eval_run_items`
- copy published dataset membership into the run transaction during `POST /api/v1/eval-runs`
- extend only `GET /api/v1/eval-runs/{run_id}` with immutable `items`
- keep create/list/retry responses as lightweight run snapshots
- render `Run items` in `/admin/eval-runs`

## Why now

The eval-run lane already explains lifecycle and retry history, but operators still cannot answer "what exactly was in this run?" from the run detail alone. A durable run-item spine is also the cleanest foundation for future per-item judging and scoring.

## Guardrails

- no judge prompts or score fields yet
- no second run-attempt model
- no heavy list payloads
- provenance must stay tenant-safe and ordered

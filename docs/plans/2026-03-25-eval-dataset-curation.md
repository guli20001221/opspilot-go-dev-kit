# 2026-03-25 Eval Dataset Curation

## Goal

Add the first incremental curation slice on top of durable eval datasets:

- `POST /api/v1/eval-datasets/{dataset_id}/items`
- `/admin/evals` -> `Add to dataset`

## Why now

The repository already has durable eval cases plus durable dataset drafts and a dataset lane.
Without an append-membership contract, datasets are still effectively create-once artifacts instead
of reusable operator-owned drafts.

## Scope

- append one durable eval case into an existing draft dataset
- keep the append idempotent for the same `dataset_id` and `eval_case_id`
- reject cross-tenant appends and non-draft dataset mutation
- let operators append from the selected eval case on `/admin/evals`

## Non-goals

- remove or reorder dataset memberships
- dataset publish/activate/archive lifecycle
- eval runs
- judge execution

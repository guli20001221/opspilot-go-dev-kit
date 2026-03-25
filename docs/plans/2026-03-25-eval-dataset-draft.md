# 2026-03-25 Eval Dataset Lane

## Goal

Add the first durable dataset-draft lane on top of the eval lane:

- `POST /api/v1/eval-datasets`
- `GET /api/v1/eval-datasets`
- `GET /api/v1/eval-datasets/{dataset_id}`
- `/admin/evals` -> `Create dataset draft`
- `/admin/eval-datasets`

## Why now

The repository now has durable promoted eval cases plus an eval queue. Without a dataset-draft contract, those promoted failures still cannot become reusable regression assets.

## Scope

- add durable dataset and membership tables
- validate tenant scope for every referenced eval case
- expose dataset create/list/get contracts
- let operators seed one dataset draft from the currently selected eval case
- add the first lightweight dataset lane and dataset-detail handoff surface

## Non-goals

- eval runs
- judge execution
- report generation from eval runs

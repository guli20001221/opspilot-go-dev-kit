# 2026-03-25 Eval Dataset Publish

## Goal
Freeze curated eval dataset drafts into immutable published baselines before later regression-run work lands.

## Scope
- add a durable `draft -> published` lifecycle transition
- record `published_by` and `published_at`
- expose `POST /api/v1/eval-datasets/{dataset_id}/publish`
- update `/admin/eval-datasets` so publish is explicit and published datasets read as immutable

## Notes
- repeated publish must return `invalid_eval_dataset_state`
- dataset membership append remains draft-only
- this slice does not add eval runs, clone, remove, or reorder flows

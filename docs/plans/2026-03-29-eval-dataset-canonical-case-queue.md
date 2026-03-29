# 2026-03-29 Eval Dataset Canonical Case Queue

## Goal
Promote dataset-wide follow-up work into canonical backend contracts so `/admin/eval-datasets` can hand operators into one dataset-scoped `/admin/cases` queue instead of only the latest eval-report queue.

## Slice
- extend `GET /api/v1/cases` with `source_eval_dataset_id`
- resolve dataset-scoped case slices through canonical eval-report lineage on the backend
- expose dataset-wide follow-up case summary and a typed dataset queue action on eval-dataset list/detail reads
- switch `/admin/eval-datasets` handoff to prefer dataset-wide case queues
- cover memory, PostgreSQL, HTTP, OpenAPI, and admin-page smoke paths

## Notes
- dataset-wide queue aggregation is additive and leaves latest-report follow-up summary intact
- the frontend should consume backend-owned dataset queue actions instead of reconstructing report IDs in browser code
- dataset summary rows should expose the same dataset-wide follow-up summary block that detail already has, so row rendering does not fall back to browser-only aggregation
- row-level case handoff should prefer the canonical dataset queue action directly and only keep latest-report queue fallback inside dataset detail where that narrower context is explicit
- dataset queue actions should prefer a direct existing-case handoff when the newest dataset-wide linked case is still open, and fall back to the dataset-scoped queue only when that newest case has already closed

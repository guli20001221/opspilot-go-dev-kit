# 2026-03-29 Eval Dataset Follow-up Triage

## Goal
Promote dataset-level unresolved regression pressure into the canonical eval-dataset list contract so `/admin/eval-datasets` can act as a real triage lane.

## Scope
- add latest eval-run linkage to `GET /api/v1/eval-datasets`
- add latest eval-report linkage to `GET /api/v1/eval-datasets`
- add `unresolved_follow_up_count` and `needs_follow_up` to dataset list rows
- add `needs_follow_up=true|false` filtering to the dataset list endpoint
- wire `/admin/eval-datasets` to the new contract with a `Needs follow-up` quick view and direct latest run/report handoff

## Notes
- keep dataset detail contract unchanged for this slice
- keep unresolved follow-up aggregation in the HTTP read-model layer for now
- avoid pushing case/report aggregation down into `internal/eval` until a stronger reuse need appears

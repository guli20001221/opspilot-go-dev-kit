## Goal

Promote unresolved eval-run follow-up pressure into the canonical `GET /api/v1/eval-runs` list contract so `/admin/eval-runs` can browse a backend-owned triage queue instead of inferring missing follow-up from detail-only reads.

## Scope

- add `item_without_open_follow_up_count` to `EvalRun`
- add `needs_follow_up` to `EvalRun`
- add `needs_follow_up=true|false` query support to `GET /api/v1/eval-runs`
- wire `/admin/eval-runs` to the new filter and unresolved queue preset
- update OpenAPI, README, architecture, runbook, and admin skill guidance

## Notes

- keep the new unresolved signal in the HTTP read-model layer for now to avoid introducing a direct `internal/eval -> internal/case` dependency
- continue to treat run detail as the heavier provenance surface and list rows as lightweight queue summaries

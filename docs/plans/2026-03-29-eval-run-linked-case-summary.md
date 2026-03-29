## Goal

Expose canonical linked-case ownership on eval-run list/detail reads so `/admin/eval-runs` can show whether follow-up work is already active before operators pivot into `/admin/cases`.

## Scope

- extend `GET /api/v1/eval-runs` and `GET /api/v1/eval-runs/{run_id}` with `linked_case_summary`
- derive latest linked case status and owner from durable case lineage on the backend
- render the summary and direct latest-case handoff on `/admin/eval-runs`
- update OpenAPI and operator docs together

## Non-goals

- changing eval-run retry or judge behavior
- adding a new eval-run-specific case queue endpoint
- moving follow-up aggregation into the browser

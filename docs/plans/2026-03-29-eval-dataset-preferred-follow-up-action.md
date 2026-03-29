## Context

`/admin/eval-datasets` can already see latest run/report linkage plus unresolved follow-up pressure from canonical dataset contracts.

The remaining gap is action routing: the page still has to decide in browser code whether the best next step is the latest eval-report unresolved bad-case queue or the latest eval-run unresolved queue.

## Decision

Add a typed `preferred_follow_up_action` to both:

- `GET /api/v1/eval-datasets`
- `GET /api/v1/eval-datasets/{dataset_id}`

The action stays backend-owned and chooses:

- `open_latest_report_queue` when the latest terminal run already materialized a durable report
- `open_latest_run_queue` when unresolved pressure exists on the latest run but no durable report exists yet
- `none` otherwise

## Outcome

`/admin/eval-datasets` can render one canonical `Open preferred queue` handoff without reconstructing run-versus-report routing logic from multiple fields.

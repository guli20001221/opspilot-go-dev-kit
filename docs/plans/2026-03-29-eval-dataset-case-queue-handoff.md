## Context

`/admin/eval-datasets` can already expose current regression pressure and the preferred evidence queue for one baseline.

The remaining gap is actual operator work: when the latest durable eval report already has open linked follow-up cases, the dataset lane should hand off directly into that canonical case queue.

## Decision

Extend canonical eval-dataset responses with latest-report case pressure:

- `open_follow_up_case_count`
- `preferred_case_queue_action`

`preferred_case_queue_action` stays read-only and backend-owned:

- `open_existing_case` when one latest follow-up case is the best drill-down
- `open_existing_queue` when there is open report-backed work but no single latest case should be highlighted
- `none` otherwise

## Outcome

`/admin/eval-datasets` can link directly to `/admin/cases` for the latest baseline follow-up queue, completing the path from baseline to evidence to current operator work.

## Context

`/admin/eval-datasets` can already see current unresolved pressure and a backend-owned preferred follow-up action.

The next operator gap is evidence: one dataset still needs a compact explanation of the latest regression attempts without forcing a jump into `/admin/eval-runs` first.

## Decision

Extend `GET /api/v1/eval-datasets/{dataset_id}` with `recent_runs[]`.

Each recent run summary includes:

- `run_id`
- `status`
- timestamps
- unresolved follow-up count
- `needs_follow_up`
- optional durable `report_id` and `report_status`

## Outcome

`/admin/eval-datasets` can show a backend-owned `Recent eval activity` panel and preserve contract-first drill-down into runs and reports.

## Goal

Move `/admin/eval-datasets` row-level primary handoff onto a backend-owned primary-action field.

## Scope

- add `preferred_primary_action` to canonical `GET /api/v1/eval-datasets` rows
- switch the main row-level button on `/admin/eval-datasets` to consume that field
- keep existing queue, report, and run links as secondary handoff surfaces

## Rule

Primary row-level handoff should prefer existing case or queue reuse before falling back to unresolved queue, report, or run drill-down.

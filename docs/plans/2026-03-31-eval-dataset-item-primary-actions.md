## Goal

Move `/admin/eval-datasets` member-level follow-up handoff onto a backend-owned primary-action field.

## Scope

- add `preferred_primary_action` to canonical `GET /api/v1/eval-datasets/{dataset_id}` `items[]`
- switch the main item-level button on `/admin/eval-datasets` to consume that field
- keep `preferred_follow_up_action` and `preferred_linked_case_action` as secondary signals

## Rule

Primary item-level handoff should prefer existing linked case or queue reuse before falling back to new case creation.

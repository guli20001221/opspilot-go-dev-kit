# 2026-03-30 Eval Run Linked Case Action

## Goal

Move `/admin/eval-runs` item-level linked-case handoff from browser-side `latest_follow_up_case_id` checks to a backend-owned typed action on the canonical eval-run detail contract.

## Scope

- add `preferred_linked_case_action` to `item_results[]` on `GET /api/v1/eval-runs/{run_id}`
- align `items[]` with the same linked-case action field
- switch `/admin/eval-runs` item and result rows to consume the canonical field
- cover the closed-latest-case edge where item-level handoff must open the queue instead of a stale case

## Notes

- `preferred_follow_up_action` still governs create-versus-reuse follow-up behavior
- `preferred_linked_case_action` only governs linked-case navigation once follow-up lineage exists
- `open_existing_queue` must appear on item-level rows when linked history exists but the latest linked case is already closed

# 2026-03-30 Eval Case Linked Case Action

## Goal

Move `/admin/evals` linked-case handoff from browser-side `latest_follow_up_case_id` branching to a backend-owned typed action on the canonical eval-case contract.

## Scope

- add `preferred_linked_case_action` to `GET /api/v1/eval-cases`
- add `preferred_linked_case_action` to `GET /api/v1/eval-cases/{eval_case_id}`
- switch `/admin/evals` list and detail linked-case handoff to consume the canonical field
- cover the closed-latest-case edge where the action must open the queue instead of a stale case

## Notes

- `preferred_follow_up_action` remains the create-versus-reuse action for opening follow-up work
- `preferred_linked_case_action` is additive and only governs linked-case navigation once follow-up exists
- closed-only history must resolve to `open_existing_queue`, not `create`

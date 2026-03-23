# 2026-03-23 Admin Taskboard Handoff Actions

## Goal

Add minimal handoff actions to the embedded task board detail panel so operators can share the exact board context or jump to the canonical task JSON without extra manual steps.

## Scope

- keep the current backend contract unchanged
- reuse the current browser URL for board-context handoff
- reuse the existing `GET /api/v1/tasks/{task_id}` endpoint for canonical task detail
- expose both actions only when a task is selected

## Implementation notes

- add `Copy task link` to copy the current task-board URL with `tenant_id`, filters, and `task_id`
- add `Open API detail` to open the existing task-detail JSON in a new tab
- keep both actions inside the existing detail action group
- do not create a second admin-only export or detail route

## Validation

- add a failing admin page HTML test for the handoff controls
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- inspect a task in the browser, confirm `Copy task link` updates the status text to a copied state, and verify `Open API detail` opens the current task JSON in a new tab

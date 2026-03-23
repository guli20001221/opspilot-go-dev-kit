# 2026-03-23 Admin Taskboard Auto Refresh

## Goal

Add a minimal operator watch mode to the embedded admin task board so task state changes can be observed without repeated manual reloads.

## Scope

- keep the current backend contract unchanged
- poll only the existing `/api/v1/admin/task-board` and `/api/v1/tasks/{task_id}` endpoints
- make auto-refresh explicit and operator-controlled
- avoid overlapping board loads while polling is active

## Implementation notes

- add a simple `Auto refresh every 5s` toggle near existing paging controls
- default the toggle to off
- skip polling when the page is hidden
- continue preserving the selected task drill-down while refreshing

## Validation

- add a failing admin page HTML test for the auto-refresh controls
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- open the task board on a waiting-approval task, enable auto-refresh, approve the task out-of-band, and verify the page updates to the terminal state without a manual reload

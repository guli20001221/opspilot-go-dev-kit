# 2026-03-23 Admin Taskboard Raw JSON

## Goal

Add a minimal raw-payload view to the embedded admin task board so operators can inspect and copy the exact single-task response without leaving the page.

## Scope

- keep the current backend contract unchanged
- reuse the existing `GET /api/v1/tasks/{task_id}` response body as-is
- show the raw JSON only inside the selected task detail panel
- provide a copy action for operator handoff and troubleshooting

## Implementation notes

- add `Show raw JSON` and `Copy raw JSON` controls alongside existing task-detail actions
- keep the raw JSON view hidden by default
- when visible, pretty-print the current detail response without inventing or flattening extra fields
- keep the copy action tied to the current selected task payload

## Validation

- add a failing admin page HTML test for the raw JSON controls
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- open a task detail in the browser, reveal the raw JSON panel, and verify the JSON contains the selected task ID and audit payload

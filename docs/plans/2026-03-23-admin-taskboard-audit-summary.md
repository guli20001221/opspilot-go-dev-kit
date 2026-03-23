# 2026-03-23 Admin Taskboard Audit Summary

## Goal

Add a compact audit-summary handoff action to the embedded task board so operators can copy a paste-ready task timeline without leaving the page.

## Scope

- keep the current backend contract unchanged
- derive the summary entirely from the current `GET /api/v1/tasks/{task_id}` response
- keep the feature inside the existing task-detail action group
- optimize for handoff text, not export formatting

## Implementation notes

- add a `Copy audit summary` control next to the existing raw JSON and link handoff actions
- include task ID, status, reason, tenant, audit ref, error reason, and audit timeline lines
- keep the source of truth in the current selected task response; do not fetch a second endpoint
- use the existing clipboard behavior pattern already established on the page

## Validation

- add a failing admin page HTML test for the audit-summary control
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- inspect a task in the browser and verify `Copy audit summary` changes to a copied state after the click

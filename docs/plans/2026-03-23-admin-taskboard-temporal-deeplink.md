# 2026-03-23 Admin Taskboard Temporal Deeplink

## Goal

Add a minimal operator deep link from the embedded admin task board into Temporal UI when a task is backed by a Temporal workflow run.

## Scope

- keep the current backend contract unchanged
- derive the deep link entirely from the existing `audit_ref`
- only surface the link in the single-task detail panel
- keep non-Temporal tasks readable by showing that no workflow link is available

## Implementation notes

- parse `audit_ref` only when it matches `temporal:workflow:<workflow_id>/<run_id>`
- point the browser link at the local Temporal UI history route
- leave approval and retry actions unchanged
- keep the board list lightweight; do not add Temporal columns to the table

## Validation

- add a failing admin page HTML test for the new Temporal execution panel
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- create a Temporal-backed report task and verify the task board detail panel opens the matching Temporal history page

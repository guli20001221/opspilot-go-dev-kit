# ADR 0003: Start Approval Workflows in API and Resume Them from the Worker

## Status

Accepted

## Context

`approved_tool_execution` now needs a real Temporal pause-and-resume path, but the repository still uses PostgreSQL task rows as the external status and audit surface.
Moving the entire approval lifecycle into Temporal-native queries and updates would be a larger contract change than this slice needs.

## Decision

For approval-gated tasks:

- the API starts a waiting Temporal workflow when the task is promoted
- the API still records approval and retry actions on the PostgreSQL task row
- the worker claims queued approval tasks from PostgreSQL and signals the existing Temporal workflow to continue
- if the approved-tool execution fails, that Temporal run closes with failure and a later retry starts a new run for the same task ID using failed-only workflow ID reuse
- PostgreSQL remains the operator-facing task state and audit surface

## Consequences

Positive:

- pause and resume are now represented in Temporal history
- approval and retry HTTP contracts remain unchanged
- the worker keeps one execution choke point for task-state progression
- failed approval runs no longer leave the worker blocked waiting on a workflow that has gone back to an idle signal wait

Trade-offs:

- the approval flow is temporarily split across API startup logic, PostgreSQL task rows, and worker-side signaling
- retry behavior is still only as strong as the current placeholder approved-tool activity path
- PostgreSQL is still the main operator surface, so Temporal visibility is additive rather than canonical

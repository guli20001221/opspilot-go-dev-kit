# ADR 0002: Bridge Report Generation Through Temporal Before Full Workflow Migration

## Status

Accepted

## Context

The repository already exposes PostgreSQL-backed task rows as the operator-facing async job surface.
We now need to start using Temporal for real execution orchestration without rewriting every task type and approval path in one step.

## Decision

For the first Temporal slice:

- keep `workflow_tasks` and `workflow_task_events` as the external task-status and audit surface
- keep the existing worker-side PostgreSQL claim loop
- route only `report_generation` through a Temporal workflow and activity
- leave approval-gated tool execution on the current non-Temporal placeholder path for now

This means the worker claims a queued report task, starts a Temporal workflow with the task ID as the business identifier, waits for completion, and then writes the terminal task status back to PostgreSQL.

## Consequences

Positive:

- lands a real Temporal-backed execution path with minimal API churn
- preserves current REST contracts and task-audit visibility
- limits migration risk to one task type

Trade-offs:

- the worker temporarily bridges two orchestration models at once: PostgreSQL claiming and Temporal execution
- approval and retry semantics are not fully Temporal-native yet
- PostgreSQL remains the source of truth for operator-visible task state until the broader migration is complete

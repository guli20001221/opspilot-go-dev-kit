# 2026-03-22 Approval Temporal Retry Recovery

## Scope

Tighten the approval-gated Temporal bridge so worker-side execution failure does not leave the workflow alive but idle, waiting for another signal while the worker blocks on `Get()`.

## Change

- make `ApprovedToolExecutionWorkflow` process a single approval or retry signal per run
- return the activity error immediately when that attempt fails
- use failed-only workflow ID reuse on retry so the same external `task_id` can start a fresh Temporal run after a failed attempt
- document the operator recovery path in the local runbook

## Why

The previous loop kept the workflow open after activity failure and returned to waiting for another signal. That made retry recovery ambiguous and risked leaving the worker blocked while the task row had not yet been marked `failed`.

## Expected outcome

- approval-task failures become visible on the PostgreSQL task row promptly
- worker-side retry becomes deterministic
- each retry gets a fresh Temporal run while preserving the stable external `task_id`

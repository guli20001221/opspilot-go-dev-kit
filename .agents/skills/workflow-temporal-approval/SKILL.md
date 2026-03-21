---
name: workflow-temporal-approval
description: Implement Temporal workflows, activities, approval gates, retries, timeouts, and async job lifecycles.
---

# workflow-temporal-approval

## Goal
Move long-running, retryable, or approval-gated work into a durable workflow layer with operator-friendly visibility.

## Use this skill when
- adding async jobs
- implementing report generation or batch evaluation
- adding approval-gated actions
- changing worker behavior, retries, or timeout strategies
- recovering from failed long-running jobs

## Inputs to collect first
- job purpose and success criteria
- sync versus async decision
- side-effect boundaries
- approval requirements
- retry and timeout expectations
- operator visibility requirements

## Likely files and directories
- `cmd/worker/**`
- `internal/workflow/**`
- `internal/storage/**`
- `docs/runbooks/**`
- workflow-related migrations or job tables

## Standard workflow
1. Decide whether the task belongs in synchronous HTTP or in a workflow.
2. Model the workflow as orchestration only.
3. Push all external I/O into activities.
4. Make activities idempotent or guard them with idempotency keys.
5. Add approval pause/resume steps for write-capable or risky operations.
6. Expose job status, error reasons, and retry affordances through APIs.
7. Test branching, approval, retry, and failure scenarios.
8. Document recovery and replay procedures in runbooks.
9. Before full Temporal execution exists, land durable task records in storage so async promotion is externally visible.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- workflow boundaries are explicit
- activities are idempotent or guarded
- approval steps are modeled, not improvised
- jobs expose status and failure details
- workflow tests cover key branches
- recovery instructions exist

## Guardrails
- no direct side effects inside workflow definitions
- no invisible retries on unsafe write operations
- no long-running task without operator visibility
- no approval-gated path without audit trail

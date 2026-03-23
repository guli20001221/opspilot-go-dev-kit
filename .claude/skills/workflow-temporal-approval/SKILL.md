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
10. If using a placeholder worker before Temporal, make task-state progression explicit and observable rather than leaving tasks stuck in queued.
11. Approval and retry endpoints should reject invalid task states explicitly rather than silently rewriting state.
12. Prefer structured task audit events over opaque audit strings when exposing operator-facing task history.
13. When task status changes and task audit events are both persisted, write them in one storage transaction rather than as separate best-effort calls.
14. During gradual migration to Temporal, it is acceptable to keep PostgreSQL task rows as the operator-facing status surface and move one task type at a time behind Temporal execution.
15. Approval-gated migrations can use the API to start a waiting workflow and the worker to signal-and-wait after approval, as long as the pause/resume path remains explicit and operator-visible.
16. If an approval-gated activity attempt fails, let that Temporal run fail and close; use retry to start a new run rather than leaving the worker blocked on a workflow that returned to waiting-for-signal.
17. For local recovery verification, prefer an explicit fault-injection toggle in activity configuration over ad-hoc code edits or hidden branch logic.
18. When an approval workflow is meant to execute a real tool after approval, persist the tool name and typed arguments on the task record so the worker activity does not have to reconstruct intent from free-form text.
19. Keep `error_reason` concise for operators, and put richer success or failure categorization into structured task audit events when you need more detail without changing the API shape.
20. When a successful workflow emits a durable artifact such as a report, persist that artifact in a separate read-model store from the worker-side success path rather than overloading the task row with artifact fields.

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

# ADR 0001: Persist workflow task records before Temporal execution

## Status

Accepted

## Context

The project already exposes async task IDs and task status APIs, but the first implementation stored those records only in process memory. That made task visibility non-durable across restarts and unusable across multiple processes.

The repository architecture calls for long-running work to move into Temporal, but landing full Temporal orchestration is larger than the immediate need to make async promotion externally visible and auditable.

## Decision

Persist workflow task records in PostgreSQL before wiring full Temporal execution.

Specifically:

- the API runtime writes promoted task records into `workflow_tasks`
- `task_id` remains the external stable identifier
- REST task lookup reads from PostgreSQL
- SSE `task_promoted` continues to emit the same `task_id`
- Temporal execution will later attach to the existing task record instead of inventing a separate external identifier

## Consequences

Positive:

- task visibility survives process restart
- REST and SSE share a single source of truth
- later Temporal integration has a stable operational record to attach to

Negative:

- workflow durability is still partial because execution is not yet Temporal-backed
- the worker process does not yet use the stored task records

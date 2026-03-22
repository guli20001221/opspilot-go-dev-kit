# Architecture summary

OpsPilot-Go is a production-oriented Golang multi-agent platform with these major layers:

- API / gateway
- application services under `internal/app/*`
- context engine
- Planner / Retrieval / Tool / Critic runtime
- workflow and approval layer
- retrieval and storage
- eval and observability
- admin console

The current foundation slice also includes a local development stack:

- PostgreSQL for application data and migrations
- Redis for future coordination and caching flows
- Temporal plus Temporal UI for workflow development
- API and worker processes bootstrapped through the same local Compose topology

The current Milestone 1 slice adds:

- `internal/session` for in-memory session and message persistence
- `internal/app/chat` as the application boundary for the synchronous chat skeleton
- `internal/contextengine` for deterministic block assembly and assembly logging
- `internal/agent/planner` for deterministic typed execution plans
- `internal/retrieval` for deterministic structured-query retrieval and provenance-bearing evidence blocks
- `internal/agent/tool` and `internal/tools/registry` for deterministic stub tool execution and approval gating
- `internal/agent/critic` for deterministic structured verdicts over draft answers, retrieval, and tool results
- `internal/workflow` for store-backed promoted task records and the current Temporal bridge layer
- `internal/storage/postgres` for the current PostgreSQL task repository and connection pool wiring
- `internal/app/httpapi` as a thin transport layer over the session and chat services
- `cmd/api` for task creation plus Temporal-backed approval-workflow initialization
- `cmd/worker` plus `internal/workflow.Runner` for PostgreSQL-backed task claiming, Temporal report execution, and Temporal approval-workflow continuation

The current synchronous chat stream now surfaces internal runtime milestones over SSE:

- `plan` when the execution plan is derived
- `retrieval` when retrieval runs
- `tool` for each executed tool step
- `task_promoted` when the internal workflow layer creates an async task

The current HTTP layer also exposes the same PostgreSQL-backed workflow records over REST:

- `POST /api/v1/tasks` for explicit async task creation
- `GET /api/v1/tasks/{task_id}` for task status lookup
- `POST /api/v1/tasks/{task_id}/approve` to resume approval-gated tasks
- `POST /api/v1/tasks/{task_id}/retry` to re-queue failed tasks
- `audit_events` embedded in task responses as the current structured operator audit view

Within the current PostgreSQL-backed workflow runtime, task-state changes and their matching
audit-event inserts now commit in the same transaction for create, claim, approve, retry,
success, and failure paths.

The current worker path advances supported queued tasks through:

- `queued -> running -> succeeded` for report generation, with the execution body now running inside a Temporal workflow and activity
- `waiting_approval -> queued -> running -> succeeded` for approved tool execution, with the waiting phase and resume signal now tracked in Temporal
- `queued -> running -> failed` for unsupported task types

This file is intentionally brief in the AI development kit.
Promote it to the main repository and expand it as implementation begins.

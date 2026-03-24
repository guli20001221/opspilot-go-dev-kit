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
- local Compose now starts prebuilt runtime images for the Go services instead of bind-mounting source and calling `go run` inside containers

The current Milestone 1 slice adds:

- `internal/session` for in-memory session and message persistence
- `internal/app/chat` as the application boundary for the synchronous chat skeleton
- `internal/app/admin/taskboard` as the first admin read model that converts workflow task pages into operator-facing task board summaries for future `web/admin` flows
- `web/admin` as the home for embedded operator pages, starting with the task board served directly by the API process
- the embedded task board page now drills into `GET /api/v1/tasks/{task_id}` for audit history and failure context instead of duplicating detail logic in the browser
- the same page now reuses `POST /api/v1/tasks/{task_id}/approve` and `POST /api/v1/tasks/{task_id}/retry` for operator actions, so the admin UI does not fork workflow mutation contracts
- the same embedded admin surface now includes `/admin/reports`, a report-lane view derived from the existing task-board and task-detail contracts rather than a new backend read model
- `internal/report` now holds the first durable report read model, emitted from successful `report_generation` workflow completion rather than inferred only from task audit history
- `internal/case` now holds the first durable operator case read model, so follow-up work can reference a source task, a source report, or both through stable IDs
- the same `internal/case` package now also supports filtered list reads for operator-facing case slices
- `internal/contextengine` for deterministic block assembly and assembly logging
- `internal/agent/planner` for deterministic typed execution plans
- `internal/retrieval` for deterministic structured-query retrieval and provenance-bearing evidence blocks
- `internal/agent/tool`, `internal/tools/registry`, and `internal/tools/http` for deterministic typed tool execution, request validation, and approval gating
- `cmd/ticketapi` plus `internal/tools/http/tickets.NewFakeHandler` for the dev-only fake ticket API used to validate the HTTP adapter path locally
- `internal/agent/critic` for deterministic structured verdicts over draft answers, retrieval, and tool results
- `internal/workflow` for store-backed promoted task records and the current Temporal bridge layer
- approval-gated workflow tasks now carry an internal tool payload for worker-side approved execution
- `internal/storage/postgres` for the current PostgreSQL task repository and connection pool wiring
- `internal/app/httpapi` as a thin transport layer over the session and chat services
- `cmd/api` for task creation plus Temporal-backed approval-workflow initialization
- `cmd/worker` plus `internal/workflow.Runner` for PostgreSQL-backed task claiming, Temporal report execution, and Temporal approval-workflow continuation and recovery
- the worker can optionally enable a dev-only approved-tool fault-injection path through configuration to verify failure and retry recovery

The current synchronous chat stream now surfaces internal runtime milestones over SSE:

- `plan` when the execution plan is derived
- `retrieval` when retrieval runs
- `tool` for each executed tool step
- `task_promoted` when the internal workflow layer creates an async task

The current HTTP layer also exposes the same PostgreSQL-backed workflow records over REST:

- `POST /api/v1/tasks` for explicit async task creation
- `GET /api/v1/tasks` for operator-facing filtered task listing with offset pagination metadata, including approval, promotion-reason, created-at, and updated-at window filters
- `GET /api/v1/tasks/{task_id}` for task status lookup
- `POST /api/v1/tasks/{task_id}/approve` to resume approval-gated tasks
- `POST /api/v1/tasks/{task_id}/retry` to re-queue failed tasks
- `GET /api/v1/admin/task-board` for the first admin read-model endpoint, which keeps visible-slice summaries and pagination metadata on the backend for future `web/admin` task pages
- `GET /admin/task-board` as the first operator page that renders summary cards, filters, and task rows directly from the admin read-model endpoint
- the board's quick-view presets now cover pending queues, queue-oriented slices, terminal success slices, report-success slices, approval-failure slices, reason slices, approval-lane slices, and task-type slices while still writing back into the same filter form and URL state
- the same page now exposes a task detail panel backed by the existing single-task API, keeping board, audit views, adjacent-task navigation, and existing task actions on one operator surface
- when a task `audit_ref` points at `temporal:workflow:<workflow_id>/<run_id>`, the detail panel now derives a direct Temporal UI history link without expanding the backend contract
- `GET /admin/reports` is the first report-focused operator page, fixed to `status=succeeded` and `task_type=report_generation` while still consuming the existing admin read-model endpoint and single-task detail endpoint
- the same report lane now keeps the selected report row visually synced with the detail pane and supports previous/next navigation within the current visible slice
- the same report lane now also supports lightweight polling against the existing admin read model, so no report-specific watch contract is needed for basic operator monitoring
- the same report lane now also supports copyable report summaries and shareable report links derived from the current task detail response, so operator handoff still reuses canonical task contracts
- `GET /api/v1/reports/{report_id}` now exposes the durable report artifact emitted by a successful report task without forcing clients to parse task audit history
- `POST /api/v1/cases` and `GET /api/v1/cases/{case_id}` now expose the durable operator case contract, separate from task/report runtime status
- `GET /api/v1/cases` now exposes the first operator-facing case list with tenant, status, source-task, and source-report filters plus offset pagination
- the same case list now supports explicit `assigned_to` filtering so queue views can map cleanly onto operator ownership without inventing frontend-only state
- `POST /api/v1/cases/{case_id}/close` now provides the first case lifecycle mutation, recording `closed_by` while keeping case status transitions explicit and REST-first
- `POST /api/v1/cases/{case_id}/assign` now provides the first case ownership mutation, recording `assigned_to` and `assigned_at` while keeping ownership explicit and REST-first
- `POST /api/v1/cases/{case_id}/notes` now provides append-only case collaboration, and `GET /api/v1/cases/{case_id}` returns recent notes without introducing a separate admin-only comment surface
- the same report lane now reads report title, summary, and readiness metadata from the durable report endpoint while still reusing task detail for audit timeline and Temporal links
- `GET /admin/cases` is the first case-focused operator page, backed directly by the durable case contract and existing task/report detail endpoints, and now supports the minimal close action for open cases
- the same case page now also supports copyable case summaries, shareable case links, and a direct jump into the canonical case-detail JSON without any admin-only debug contract
- the same case page now also surfaces and updates assignment, so ownership stays in the canonical case contract instead of drifting into frontend-only state
- the same case page now also shows and appends recent notes, so operator handoff context lives on the case instead of being implied by task/report provenance
- the same case page now defaults into an open-case queue view, adds a `My open cases` shortcut, and computes age/staleness from canonical `updated_at`
- the existing task-board and report-lane detail panes can now create durable cases by reusing `POST /api/v1/cases`, keeping case creation on canonical task/report surfaces instead of inventing admin-only write APIs
- the same report lane can now surface and copy the raw durable report JSON directly from the report endpoint, so artifact troubleshooting stays contract-first too
- the same report lane now falls back to task provenance when a legacy or partially recovered successful report task has no durable report row, so operator drill-down remains readable
- the worker now finalizes report success and durable report persistence together, so report `ready_at` and `metadata.audit_ref` match the final task success surface
- the same page can optionally auto-refresh against the existing board and task-detail endpoints, so operator monitoring does not require manual reload loops
- the board now also includes quick-view presets for common operator slices, but those presets still flow through the same existing filter fields and backend read model
- the detail panel can now expose the raw single-task JSON payload from the existing detail endpoint, keeping debugging and escalation views contract-first as well
- the same detail panel now exposes handoff actions that stay contract-first: copy the selected board URL or open the canonical task-detail JSON in a separate tab
- the same detail panel can now derive a compact audit-summary string from the selected task response and its audit events, giving operators a contract-first handoff artifact without a new backend surface
- the same detail panel now supports previous/next navigation within the current board slice and derives execution/timeline digest cards from the selected task response without introducing extra backend aggregation
- the same detail panel can now reapply the board filters to the selected task lane by writing back into the existing tenant/task-type/reason/requires-approval filters rather than introducing a separate frontend query model
- the same detail panel can now also reapply the selected task queue back into the existing board filter form by combining `status` and `requires_approval`, keeping queue-based triage inside the same query model
- the same detail panel can now also reapply the selected task type back into the existing board filter form, keeping task-type triage inside the same query model
- the same detail panel can now also reapply the selected task approval lane back into the existing board filter form, keeping approval-lane triage inside the same query model
- the same detail panel can now also reapply the selected task reason back into the existing board filter form, keeping reason-based triage inside the same query model
- the board now also keeps the selected row highlighted and scroll-synced with the detail panel, so navigation across the current slice does not break table context
- the same detail panel can also reapply the selected task status back into the existing board filter form, keeping status-based triage inside the same query model
- `audit_events` embedded in task responses as the current structured operator audit view
- the list endpoint intentionally omits `audit_events`, so the summary surface stays cheap while the single-task endpoint remains the detailed audit drill-down, and it returns `has_more` plus `next_offset` for simple operator pagination
- `error_reason` normalized to an operator-facing summary while deep Temporal detail remains in worker logs

Within the current PostgreSQL-backed workflow runtime, task-state changes and their matching
audit-event inserts now commit in the same transaction for create, claim, approve, retry,
success, and failure paths.

The current worker path advances supported queued tasks through:

- `queued -> running -> succeeded` for report generation, with the execution body now running inside a Temporal workflow and activity
- `waiting_approval -> queued -> running -> succeeded` for approved tool execution, with the waiting phase and resume signal tracked in Temporal
- `waiting_approval -> queued -> running -> failed -> queued -> running -> succeeded` for approved tool execution recovery, where a failed approval run closes and retry starts a new Temporal run for the same task ID
- approval tasks promoted from chat carry the selected tool name and typed arguments; legacy tasks without payload keep the older placeholder-compatible execution path
- the default ticket adapters now execute through typed request/response contracts, so approved-tool runs can reject invalid payloads instead of silently succeeding on fixed stub output
- registry construction is now config-driven: without a ticket API base URL it uses deterministic local adapters, and with one it switches both API and worker to the HTTP ticket adapter through the same typed executor hook
- task success audit events now carry execution summaries from the executor path, which gives operators a concise description of what completed without changing the task response schema
- task failure audit events now use categorized detail prefixes while leaving `error_reason` as the shorter root-cause string
- `queued -> running -> failed` for unsupported task types

This file is intentionally brief in the AI development kit.
Promote it to the main repository and expand it as implementation begins.

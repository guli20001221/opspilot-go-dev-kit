# OpsPilot-Go AI development kit

This package contains the repository-level AI instructions for building OpsPilot-Go with Codex and Claude Code.

Included:
- final `AGENTS.md`
- `CLAUDE.md` wrapper
- `docs/document-governance.md` for source-of-truth order and conflict handling
- 12 Claude-native skills under `.claude/skills/`
- recommended local `AGENTS.override.md` files for key subsystems
- support READMEs for agents, hooks, ADRs, and runbooks
- a complete recommended repository tree

Use this package as the governance and playbook layer for your main application repository.

Current foundation slice:
- `go.mod` with the initial Go module bootstrap
- `cmd/api` serving `/healthz` and `/readyz`
- `cmd/worker` process bootstrap and graceful shutdown wiring
- `cmd/ticketapi` for a dev-only fake ticket API used by the local compose stack
- shared config and `slog` logging packages under `internal/app`
- a first SQL migration scaffold under `db/migrations`
- `compose.yaml` for local PostgreSQL, Redis, Temporal, API, and worker bootstrapping
- the local compose stack now also includes a fake ticket API so the configurable HTTP ticket adapters can be exercised end-to-end without an external system
- the local compose stack now builds dedicated runtime images for `api`, `worker`, and `ticket-api`, so container start no longer depends on runtime `go run` downloads
- API container published on host port `18080` to avoid common local `8080` conflicts
- `Makefile` targets for `fmt`, `test`, `build`, and `check`
- `scripts/dev/tasks.ps1` as the verified PowerShell fallback when `make` is unavailable
- local bootstrap instructions in `docs/runbooks/local-bootstrap.md`
- static OpenAPI contract under `docs/openapi/openapi.yaml`

Current Milestone 1 slice:
- in-memory session and message persistence under `internal/session`
- typed chat application service under `internal/app/chat`
- `internal/app/admin/taskboard` as the first admin-facing task board read model, with visible-slice status and reason summaries for future `web/admin` task views
- `web/admin` now ships the first embedded operator task board page, served by the API at `/admin/task-board`
- the embedded admin task board page now supports in-page single-task drill-down using the existing `GET /api/v1/tasks/{task_id}` detail contract
- the same page now exposes `approve` and `retry` controls in the detail panel by reusing the existing task action endpoints instead of adding admin-only mutation APIs
- the same detail panel now derives a Temporal workflow history deep link from `audit_ref` when the task is running on a Temporal-backed execution path
- the board now offers an optional 5-second auto-refresh mode so operators can watch task transitions without manually reloading the page
- the board now also offers quick-view presets for common operator slices such as `Needs approval`, `Failed`, and `Running`
- the task detail panel now includes a raw JSON view and copy action for direct operator troubleshooting without leaving the page
- the same task detail panel now supports handoff actions: copy the current task-board URL and open the underlying task API detail in a new tab
- the same task detail panel now supports `Copy audit summary` for a compact, paste-ready task timeline handoff
- the same detail panel now supports visible-slice task navigation plus digest cards for execution summary and timeline state, so operators can triage adjacent tasks without bouncing back to the table
- deterministic context assembly under `internal/contextengine`
- deterministic typed planning under `internal/agent/planner`
- deterministic typed retrieval under `internal/retrieval`
- deterministic typed tool execution under `internal/agent/tool`, `internal/tools/registry`, and `internal/tools/http`
- deterministic typed critic review under `internal/agent/critic`
- PostgreSQL-backed async promotion records under `internal/workflow` for the API runtime
- worker-side task progression from `queued` to `running/succeeded/failed`, with `report_generation` bridged through Temporal workflow execution
- approval-gated `approved_tool_execution` tasks now start a waiting Temporal workflow at promote time, fail the current Temporal run on execution error, and use retry to start a new failed-only Temporal run for the same task
- `POST /api/v1/sessions` for session creation
- `GET /api/v1/sessions/{session_id}/messages` for message listing
- `POST /api/v1/tasks` for PostgreSQL-backed task creation
- `GET /api/v1/tasks` for operator-facing task listing with `tenant_id`, `status`, `task_type`, `reason`, `requires_approval`, `created_after`, `created_before`, `updated_after`, `updated_before`, `limit`, and `offset` filters
- `GET /api/v1/tasks/{task_id}` for persisted task status lookup
- `POST /api/v1/tasks/{task_id}/approve` and `POST /api/v1/tasks/{task_id}/retry` for minimal task actions
- `GET /api/v1/admin/task-board` for the first backend task-board read model that returns items, page metadata, and visible-slice summary counts for future `web/admin` task views
- `GET /admin/task-board` for the first embedded operator page consuming the backend task-board read model
- the admin page keeps the board summary lightweight while letting operators inspect per-task audit history, navigate adjacent visible tasks, and trigger existing task actions from one detail panel
- Temporal-backed tasks now expose a direct workflow-history deep link in that same detail panel, so operators can jump from the board into Temporal UI without a second lookup step
- the same page can now poll the existing board and task-detail endpoints every 5 seconds when the operator enables auto-refresh
- common operator slices can now be applied from quick-view buttons instead of manually composing the same filters each time
- the detail panel can now reveal the full single-task JSON payload and copy it to the clipboard for debugging and escalation flows
- the same panel now also supports direct handoff into the canonical task detail URL, either by copying the current board link or opening the underlying API JSON directly
- operators can also copy a compact audit summary derived from the current detail response and its audit timeline for incident notes or handoff messages
- structured `audit_events` on task responses for create, claim, approve, retry, succeed, and fail
- list-task responses intentionally omit `audit_events` so the operator list view stays lightweight while single-task lookup remains the detailed drill-down surface, and now return `has_more` plus `next_offset` for simple offset pagination
- workflow task row changes and matching `audit_events` now commit atomically in the PostgreSQL-backed runtime paths
- successful task audit events now carry an operator-facing execution summary, so approved-tool tasks show what action completed instead of only `succeeded`
- failed task audit events now carry categorized detail strings such as `validation_error:` or `authorization_error:` while `error_reason` stays as the shorter root-cause summary
- the local worker uses Temporal for `report_generation` while keeping PostgreSQL task rows as the current operator-facing status surface
- the local API also uses a Temporal client to initialize waiting approval workflows for `approved_tool_execution`, while worker-side retry uses Temporal failed-only ID reuse for recovery
- the local worker also supports a dev-only `OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE` toggle so the approval failure and retry path can be verified end-to-end without changing public APIs
- failed task `error_reason` values are now normalized to short operator-facing summaries instead of full wrapped Temporal error strings
- approval tasks promoted from chat now persist an internal tool payload so the Temporal approved-tool activity can execute a typed registered tool after approval instead of always using a placeholder path
- the default ticket tools now validate request payloads and return argument-dependent structured results through deterministic typed adapters instead of fixed stub JSON
- if `OPSPILOT_TICKET_API_BASE_URL` is configured, the API and worker switch the default ticket tools from deterministic local adapters to a real HTTP boundary while preserving the same internal tool contracts
- `POST /api/v1/chat/stream` with optional SSE `plan`, `retrieval`, `tool`, and `task_promoted` events ahead of `state -> done`

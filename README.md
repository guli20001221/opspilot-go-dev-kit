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
- shared config and `slog` logging packages under `internal/app`
- a first SQL migration scaffold under `db/migrations`
- `compose.yaml` for local PostgreSQL, Redis, Temporal, API, and worker bootstrapping
- API container published on host port `18080` to avoid common local `8080` conflicts
- `Makefile` targets for `fmt`, `test`, `build`, and `check`
- `scripts/dev/tasks.ps1` as the verified PowerShell fallback when `make` is unavailable
- local bootstrap instructions in `docs/runbooks/local-bootstrap.md`
- static OpenAPI contract under `docs/openapi/openapi.yaml`

Current Milestone 1 slice:
- in-memory session and message persistence under `internal/session`
- typed chat application service under `internal/app/chat`
- deterministic context assembly under `internal/contextengine`
- deterministic typed planning under `internal/agent/planner`
- deterministic typed retrieval under `internal/retrieval`
- deterministic typed tool execution under `internal/agent/tool` and `internal/tools/registry`
- deterministic typed critic review under `internal/agent/critic`
- PostgreSQL-backed async promotion records under `internal/workflow` for the API runtime
- worker-side task progression from `queued` to `running/succeeded/failed`, with `report_generation` bridged through Temporal workflow execution
- approval-gated `approved_tool_execution` tasks now start a waiting Temporal workflow at promote time, fail the current Temporal run on execution error, and use retry to start a new failed-only Temporal run for the same task
- `POST /api/v1/sessions` for session creation
- `GET /api/v1/sessions/{session_id}/messages` for message listing
- `POST /api/v1/tasks` for PostgreSQL-backed task creation
- `GET /api/v1/tasks/{task_id}` for persisted task status lookup
- `POST /api/v1/tasks/{task_id}/approve` and `POST /api/v1/tasks/{task_id}/retry` for minimal task actions
- structured `audit_events` on task responses for create, claim, approve, retry, succeed, and fail
- workflow task row changes and matching `audit_events` now commit atomically in the PostgreSQL-backed runtime paths
- the local worker uses Temporal for `report_generation` while keeping PostgreSQL task rows as the current operator-facing status surface
- the local API also uses a Temporal client to initialize waiting approval workflows for `approved_tool_execution`, while worker-side retry uses Temporal failed-only ID reuse for recovery
- `POST /api/v1/chat/stream` with optional SSE `plan`, `retrieval`, `tool`, and `task_promoted` events ahead of `state -> done`

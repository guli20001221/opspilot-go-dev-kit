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
- deterministic async promotion records under `internal/workflow`
- `POST /api/v1/sessions` for session creation
- `GET /api/v1/sessions/{session_id}/messages` for message listing
- `POST /api/v1/chat/stream` with a thin HTTP handler delegating to SSE `meta -> state -> done` placeholder flow

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
- `Makefile` targets for `fmt`, `test`, `build`, and `check`
- `scripts/dev/tasks.ps1` as the verified PowerShell fallback when `make` is unavailable
- local bootstrap instructions in `docs/runbooks/local-bootstrap.md`

---
name: repo-bootstrap-go
description: Bootstrap or refactor the OpsPilot-Go repository skeleton, local developer workflow, and minimum runnable foundation.
---

# repo-bootstrap-go

## Goal
Create or realign the repository to a clean modular-monolith Golang layout with reproducible local setup, clear entrypoints, and minimal operational hygiene.

## Use this skill when
- starting a fresh repository
- restructuring a drifted repository
- adding foundational files such as `Makefile`, local compose, config loading, health endpoints, or bootstrap docs
- preparing the repo for other skills to land cleanly

## Inputs to collect first
- current repository tree
- required entrypoints (`cmd/api`, `cmd/worker`)
- local stack expectations (Postgres, Redis, Temporal, observability)
- preferred Go version and module name
- whether frontend lives in the same repo

## Likely files and directories
- `go.mod`, `go.sum`
- `cmd/api/**`, `cmd/worker/**`
- `internal/app/**`
- `config/**`
- `Makefile`
- `docker/**` or `compose.yaml`
- `.env.example`
- `README.md`
- `docs/runbooks/local-bootstrap.md`

## Standard workflow
1. Inspect the current tree and compare it with the target repository shape from `AGENTS.md`.
2. Keep only the smallest necessary directories to make the repo runnable.
3. Create or update:
   - module metadata
   - configuration loading
   - structured logging
   - health and readiness endpoints
   - make targets for fmt, lint, test, dev-up, and dev-down
4. Add the minimum local stack definition for app dependencies.
5. Add entrypoints that compile even if business logic is still stubbed.
6. Update README and local runbooks with tested commands.
7. Leave clear TODO boundaries instead of half-implemented infrastructure.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- repository skeleton matches the agreed shape
- main entrypoints build
- local bootstrap commands exist and are documented
- health endpoints or equivalent readiness checks exist
- README reflects actual commands
- follow-up gaps are explicit, not hidden

## Guardrails
- do not introduce microservices without approval
- do not add heavy frameworks just to scaffold quickly
- do not create dead directories with no ownership or purpose
- do not claim local setup works unless you ran the bootstrap commands

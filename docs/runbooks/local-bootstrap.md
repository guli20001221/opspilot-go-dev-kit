# Local Bootstrap

## Scope

This runbook covers the current foundation slice only:

- Go module bootstrap
- API binary with `/healthz` and `/readyz`
- worker bootstrap
- local Docker Compose stack for PostgreSQL, Redis, Temporal, API, and worker
- Make targets for format, test, build, and check

It does not yet wire real DB access from the app code or a real OpenTelemetry exporter.

## Prerequisites

- Go 1.24.2
- Optional: `make`
- PowerShell for the fallback script on Windows
- Docker Desktop with the daemon running

## Commands

1. Copy `.env.example` values into your local shell environment if you need overrides.
2. If `make` is installed, run `make test` and `make build`.
3. If `make` is not installed, run `powershell -File scripts/dev/tasks.ps1 test` and `powershell -File scripts/dev/tasks.ps1 build`.
4. Validate the Compose file with `docker compose config`.
5. Start the local stack with `make dev-up` or `powershell -File scripts/dev/tasks.ps1 dev-up`.
6. Check `http://localhost:18080/healthz`.
7. Check `http://localhost:18080/readyz`.
8. Check Temporal UI at `http://localhost:8088`.

Successful build artifacts are emitted under `bin/`.

## Current API surface

- `POST /api/v1/sessions`
- `GET /api/v1/sessions/{session_id}/messages`
- `POST /api/v1/chat/stream`

The current chat stream implementation is a Milestone 1 skeleton:
- session storage is in-memory
- SSE events are limited to `meta`, `state`, and `done`
- assistant output is a fixed placeholder response

## Current gaps

- In the current Windows shell, `make` may be unavailable; use `scripts/dev/tasks.ps1` as the verified fallback.
- The application does not yet open PostgreSQL, Redis, or Temporal connections.
- Only the first bootstrap SQL migration exists.
- No trace exporter exists yet; only request-scoped IDs are logged.

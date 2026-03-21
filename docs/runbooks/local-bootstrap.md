# Local Bootstrap

## Scope

This runbook covers the current foundation slice only:

- Go module bootstrap
- API binary with `/healthz` and `/readyz`
- worker bootstrap
- Make targets for format, test, build, and check

It does not yet provision PostgreSQL, Redis, Temporal, or OpenTelemetry exporters.

## Prerequisites

- Go 1.24.2
- Optional: `make`
- PowerShell for the fallback script on Windows

## Commands

1. Copy `.env.example` values into your local shell environment if you need overrides.
2. If `make` is installed, run `make test` and `make build`.
3. If `make` is not installed, run `powershell -File scripts/dev/tasks.ps1 test` and `powershell -File scripts/dev/tasks.ps1 build`.
4. Start the API with `go run ./cmd/api`.
5. Check `http://localhost:8080/healthz`.
6. Check `http://localhost:8080/readyz`.
7. Start the worker with `go run ./cmd/worker`.

Successful build artifacts are emitted under `bin/`.

## Current gaps

- `make dev-up` and `make dev-down` are intentionally placeholders.
- In the current Windows shell, `make` may be unavailable; use `scripts/dev/tasks.ps1` as the verified fallback.
- No database migrations exist yet.
- No Redis or Temporal worker wiring exists yet.
- No trace exporter exists yet; only request-scoped IDs are logged.

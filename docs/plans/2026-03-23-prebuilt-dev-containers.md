# 2026-03-23 Prebuilt Dev Containers

## Goal

Stabilize the local Compose stack by removing runtime `go run` execution from the `api`, `worker`, and `ticket-api` containers.

## Scope

- add dedicated Dockerfiles for `cmd/api`, `cmd/worker`, and `cmd/ticketapi`
- switch `compose.yaml` to `build:`-based services that run compiled binaries
- add `.dockerignore`
- update `make dev-up` and the PowerShell fallback to build before starting
- update setup docs to reflect the new startup path

## Key decisions

- use simple multi-stage Dockerfiles instead of bind-mounted source plus runtime `go run`
- keep the rest of the local dependency graph unchanged: PostgreSQL, Redis, Temporal, and Temporal UI stay as they are
- avoid extra build frontends or runtime package managers so the local stack only needs base images plus normal `docker compose build`

## Validation

- `docker compose config`
- `docker compose build ticket-api api worker`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

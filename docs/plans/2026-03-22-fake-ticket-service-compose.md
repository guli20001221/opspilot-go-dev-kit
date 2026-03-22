# 2026-03-22 Fake Ticket Service Compose

## Goal
Add an in-repo fake ticket API to the local development stack so the configurable HTTP ticket adapter path can be verified end-to-end without an external dependency.

## Scope
- add a dev-only fake ticket HTTP handler
- add a `cmd/ticketapi` entrypoint
- wire the fake service into `compose.yaml`
- route compose-managed API and worker processes through that fake service by default

## Non-goals
- real ticket system behavior
- new public APIs
- production deployment wiring

## Validation
- handler unit tests for search, comment creation, and auth
- `docker compose config`
- local compose smoke test through `chat -> approval task -> fake ticket API`
- repo `check` command

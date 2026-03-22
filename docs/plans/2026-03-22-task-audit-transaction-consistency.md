# 2026-03-22 Task Audit Transaction Consistency

## Goal

Keep workflow task state and structured `audit_events` consistent by committing them together in the PostgreSQL-backed runtime paths.

## Scope

- add combined store methods for create and update plus matching event append
- move worker claim auditing into the same store transaction as `queued -> running`
- switch workflow service and runner paths away from separate best-effort event writes
- cover PostgreSQL behavior with integration tests

## Non-goals

- Temporal SDK integration
- audit-event search endpoints
- retry backoff or retry limits

## Validation

- `go test ./internal/storage/postgres -count=1`
- `go test ./internal/workflow -count=1`
- `go test ./...`
- runtime verification that task creation and worker progression still surface ordered `audit_events`

# 2026-03-22 Task Audit Events

## Goal

Expose a structured audit trail for task lifecycle transitions instead of relying only on the latest `audit_ref` summary field.

## Scope

- add `workflow_task_events` migration
- append audit events for create, claim, approve, retry, succeed, and fail
- surface `audit_events` on task responses
- cover memory store and PostgreSQL store behavior with tests

## Non-goals

- full audit search endpoints
- cross-task reporting
- immutable actor identity verification

## Validation

- `go test ./internal/workflow -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./internal/storage/postgres -count=1`
- `go test ./...`
- runtime verification that task responses include ordered audit events

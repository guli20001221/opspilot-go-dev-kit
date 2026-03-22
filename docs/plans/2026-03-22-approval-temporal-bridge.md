# 2026-03-22 Approval Temporal Bridge

## Goal

Move the waiting and resume path of `approved_tool_execution` into Temporal while preserving the existing REST task surface.

## Scope

- start a waiting Temporal workflow when an approval-gated task is promoted
- route claimed approved-tool tasks through a Temporal signal-and-wait executor
- keep approval and retry endpoints on the existing PostgreSQL-backed task API
- update local compose wiring so both API and worker have Temporal access

## Non-goals

- replacing PostgreSQL task rows as the operator-facing status surface
- converting all retries into fully Temporal-native task queries
- adding new public endpoints

## Validation

- `go test ./internal/workflow -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- `docker compose up -d --build api worker`
- runtime verification that approved-tool tasks still reach `succeeded` after approval

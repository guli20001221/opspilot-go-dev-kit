# 2026-03-22 Temporal Report Workflow

## Goal

Move the first real task type, `report_generation`, behind Temporal workflow execution while preserving the current PostgreSQL-backed task API.

## Scope

- add Temporal config for the worker runtime
- add a Temporal-aware executor that routes report tasks to a report workflow
- register a minimal report workflow and activity on the worker
- keep approved tool execution on the existing fallback path
- document the bridge architecture and local setup expectations

## Non-goals

- full approval flow migration into Temporal
- task queries against Temporal directly
- replacing PostgreSQL task rows as the operator-facing task surface

## Validation

- `go test ./internal/app/config -count=1`
- `go test ./internal/workflow -count=1`
- `go test ./...`
- `docker compose up -d --build api worker`
- runtime verification that a `report_generation` task still reaches `succeeded`

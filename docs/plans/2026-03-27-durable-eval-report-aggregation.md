# 2026-03-27 Durable Eval Report Aggregation

## Goal
Materialize each completed eval run into a durable aggregated backend artifact instead of recomputing operator metrics from raw run detail on every read.

## Scope
- add a canonical `EvalReport` read model inside `internal/eval`
- aggregate totals, average score, judge metadata, and bad-case lineage from terminal `item_results`
- persist the artifact in both memory and PostgreSQL stores
- materialize the report automatically from the eval worker after terminal run finalization

## Non-goals
- no new public API contract yet
- no new admin UI yet
- no baseline comparison or report diff surface yet

## Validation
- targeted `go test ./internal/eval ./cmd/worker`
- `go test ./...`
- targeted PostgreSQL store round-trip when a local DSN is available

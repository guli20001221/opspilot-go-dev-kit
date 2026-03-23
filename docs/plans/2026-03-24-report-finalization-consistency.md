# 2026-03-24 report finalization consistency

## Goal
Close correctness gaps between workflow task success and durable report persistence.

## Decision
- finalize successful report tasks and durable report rows together in one PostgreSQL transaction
- build report metadata from the final successful task state, including the final `audit_ref`
- derive `ready_at` from the final successful task timestamp
- let `/admin/reports` fall back to task provenance when a successful report task has no durable report row

## Validation
- targeted workflow/report/postgres/httpapi tests
- full `go test ./...`
- repo `check` target
- smoke test for report/task consistency
- browser smoke for missing durable report row fallback

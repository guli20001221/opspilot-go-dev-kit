# 2026-03-27 Eval Run Item Results

## Goal
Extend the canonical eval-run detail contract with durable placeholder per-item terminal outcomes while keeping create, list, and retry responses lightweight.

## Scope
- add durable storage for eval-run item results
- write placeholder per-item results on run success and failure
- clear stale per-item results when retry re-queues the same durable run
- expose item results only on `GET /api/v1/eval-runs/{run_id}`
- render item results on `/admin/eval-runs`

## Non-goals
- judge prompts
- score aggregation
- per-item retries
- list-level result summaries

## Validation
- targeted Go tests for `internal/eval`, `internal/app/httpapi`, `internal/storage/postgres`, and `cmd/worker`
- full `go test ./...`
- OpenAPI YAML parse check
- repo `check` script

# 2026-03-27 Eval Run Placeholder Judge

## Goal
Replace string-only placeholder eval item results with a minimal structured judge contract that can survive into a later provider-backed judge implementation.

## Scope
- add structured placeholder judge fields to durable eval-run item results
- persist verdict, score, judge version, and raw judge output
- expose those fields through `GET /api/v1/eval-runs/{run_id}`
- render the structured fields in `/admin/eval-runs`

## Non-goals
- external model-provider judge calls
- judge prompt management
- aggregate rubric scoring
- regression report comparison changes

## Validation
- targeted `go test ./internal/eval ./internal/app/httpapi ./internal/storage/postgres -count=1`
- full `go test ./...`
- OpenAPI YAML parse check
- repo `check` script

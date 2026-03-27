# 2026-03-27 Eval Run Result Summary

## Goal
Expose lightweight terminal pass/fail counts on canonical eval-run reads so operators can scan outcomes without expanding the full per-item result payload.

## Scope
- attach `result_summary` to canonical eval-run reads for terminal runs
- keep `create`, queued list rows, running list rows, and retry responses lightweight
- preserve `item_results` as the heavier per-item detail payload on run detail
- render the summary counts in `/admin/eval-runs`

## Non-goals
- judge scoring
- aggregate score rubrics
- dataset-level result rollups
- new admin-only eval-run contracts

## Validation
- targeted Go tests for `internal/eval`, `internal/app/httpapi`, and `internal/storage/postgres`
- full `go test ./...`
- OpenAPI YAML parse check
- repo `check` script

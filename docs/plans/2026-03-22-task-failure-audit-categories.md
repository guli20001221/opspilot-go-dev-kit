# 2026-03-22 Task Failure Audit Categories

## Goal
Improve operator debugging by classifying failed task audit details without changing the public task schema or replacing the existing short `error_reason`.

## Scope
- classify failed task audit detail strings into coarse categories
- keep `error_reason` as the concise root-cause summary
- validate the behavior through workflow tests, task API tests, and a local compose failure smoke test

## Non-goals
- new task response fields
- a full error taxonomy model in storage
- changing Temporal retry behavior

## Validation
- focused workflow and HTTP API tests
- full `go test ./...`
- repo `check` command
- compose smoke test showing a categorized failure detail

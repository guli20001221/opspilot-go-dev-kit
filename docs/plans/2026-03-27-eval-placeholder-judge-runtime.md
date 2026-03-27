# 2026-03-27 Eval Placeholder Judge Runtime

## Goal
Extract the built-in placeholder eval judge behind an explicit runtime boundary so later provider-backed judging can replace it without redesigning the run-result contract.

## Scope
- add a typed `RunJudge` interface under `internal/eval`
- move placeholder verdict/score/raw-output construction out of `run_service.go`
- pin the built-in judge to a stable version ID and prompt artifact path
- keep the current worker behavior deterministic and credential-free

## Non-goals
- external model/provider integration
- score aggregation
- new eval APIs
- admin-only judge controls

## Validation
- targeted `go test ./internal/eval`
- full `go test ./...`
- repo `check` script

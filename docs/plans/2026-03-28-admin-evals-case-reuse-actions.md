# 2026-03-28 admin evals case reuse actions

## Goal
Make canonical eval-case follow-up reuse visible inside `/admin/evals` before the operator triggers another case-creation write.

## Scope
- switch the eval-case primary action from `Create case` to `Open existing case` or `Open existing queue` when the selected eval case already has open follow-up
- keep the backend contract unchanged and reuse the existing canonical case IDs and queue links
- add a runtime smoke covering the dynamic button state when a durable eval case already has open follow-up work

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

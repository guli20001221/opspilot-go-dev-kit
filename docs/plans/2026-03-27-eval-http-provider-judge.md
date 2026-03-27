# 2026-03-27 Eval HTTP Provider Judge

## Goal
Add the first provider-backed eval judge execution path behind the existing `RunJudge` boundary without changing the durable eval-run contract.

## Scope
- add env-gated judge provider selection in the worker
- add a generic HTTP JSON eval judge implementation
- keep the placeholder judge as the default local path
- preserve canonical `item_results` shape for both placeholder and provider paths
- fall back to placeholder failed results if the external judge call fails during run finalization

## Non-goals
- no public eval API changes
- no new admin UI
- no provider-specific SDK dependency
- no live credential requirement for local development

## Validation
- targeted `go test ./internal/eval ./internal/app/config`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

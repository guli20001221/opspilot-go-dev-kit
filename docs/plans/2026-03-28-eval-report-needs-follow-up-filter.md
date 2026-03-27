# 2026-03-28 Eval Report Needs-Follow-Up Filter

## Goal

Let operators pull unresolved eval regressions directly from the canonical eval-report list contract and expose that slice on `/admin/eval-reports`.

## Scope

- add `needs_follow_up=true|false` to `GET /api/v1/eval-reports`
- implement filter semantics from existing follow-up case summary data
- add a `Needs follow-up` quick view to `/admin/eval-reports`
- sync tests, OpenAPI, docs, and admin/eval skills

## Non-goals

- no new pages
- no new tables
- no detail-endpoint changes
- no case write-surface changes

## Validation

- targeted `go test ./internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI YAML parse check

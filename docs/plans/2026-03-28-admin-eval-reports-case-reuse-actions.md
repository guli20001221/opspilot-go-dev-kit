# 2026-03-28 admin eval reports case reuse actions

## Goal
Make canonical case reuse visible inside `/admin/eval-reports` before the operator triggers another follow-up write.

## Scope
- switch the report-level primary action from `Create case` to `Open existing case` or `Open existing queue` when the selected eval report already has open follow-up
- switch bad-case row actions from `Create case from bad case` to `Open existing bad-case case` or `Open bad-case queue` when that bad case already has open follow-up
- keep the backend contract unchanged and reuse the existing canonical case IDs and queue links

## Validation
- targeted `go test` for `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

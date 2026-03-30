# 2026-03-30 Eval Report Bad-Case Primary Actions

## Goal

Move `/admin/eval-reports` bad-case primary actions from browser-side heuristics to a backend-owned canonical field.

## Scope

- add `preferred_primary_action` to `GET /api/v1/eval-reports/{report_id}` bad-case rows
- update `/admin/eval-reports` bad-case buttons to consume that field
- keep `preferred_linked_case_action` as the secondary latest-case or queue handoff
- update OpenAPI, docs, and skill guidance

## Validation

- focused `go test` for eval-report HTTP and admin page smoke
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-28 Admin Eval Reports Linked Cases

## Goal

Close the operator loop from a durable eval report to the durable follow-up cases already linked to it.

## Scope

- reuse `GET /api/v1/cases?tenant_id=...&source_eval_report_id=...`
- show a small linked-case summary inside `/admin/eval-reports` detail
- add a direct `Open linked cases` handoff back to `/admin/cases`
- keep the slice read-only and contract-first
- sync docs and skill guidance

## Non-goals

- no new case API
- no new eval-report API fields
- no write actions from `/admin/eval-reports`

## Validation

- targeted `go test ./internal/app/httpapi -run 'TestAdminEvalReportsPageRendersHTML|TestAdminEvalReportsPageRuntimeSmoke'`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

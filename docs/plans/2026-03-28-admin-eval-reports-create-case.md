# 2026-03-28 Admin Eval Reports Create Case

## Goal
Let operators create a durable follow-up case directly from `/admin/eval-reports`.

## Scope
- add `Create case` to the eval-report detail pane
- reuse canonical `POST /api/v1/cases`
- seed the request with `source_eval_report_id`
- deep-link to `/admin/cases?case_id=...` after creation
- keep report detail and linked-case reads on existing canonical endpoints

## Verification
- `go test ./internal/app/httpapi -run 'TestAdminEvalReportsPageRendersHTML|TestAdminEvalReportsPageRuntimeSmoke' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

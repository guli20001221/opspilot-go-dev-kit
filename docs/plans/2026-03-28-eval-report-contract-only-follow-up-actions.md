# 2026-03-28 Eval Report Contract-Only Follow-Up Actions

## Goal
Remove browser-side count heuristics from `/admin/eval-reports` once canonical follow-up action fields exist on the report detail and bad-case detail contracts.

## Why
- report-level and bad-case-level follow-up actions already have typed backend-owned fields
- the page still carried duplicate count-based fallback logic
- keeping both paths alive invites drift between operator UI and canonical API behavior

## Slice
1. remove count-based fallback from report-level `preferredFollowUpAction(item)`
2. remove count-based fallback from bad-case-level `badCasePrimaryAction(item, badCase)`
3. keep the default safe behavior as `create` when the typed field is absent
4. sync the admin-console skill so future slices do not reintroduce browser-side heuristics

## Validation
- `go test ./internal/app/httpapi -run 'TestAdminEvalReportsPageRendersHTML|TestAdminEvalReportsPageRuntimeSmoke' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

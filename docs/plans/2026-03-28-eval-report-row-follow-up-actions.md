# 2026-03-28 Eval Report Row Follow-Up Actions

## Goal
Expose row-level follow-up handoff on `/admin/eval-reports` using the canonical `preferred_follow_up_action` already present on eval-report list rows.

## Why
- detail already supports create-versus-reuse through backend-owned action fields
- operators still had to open detail before acting on report-level follow-up
- the list contract already carries enough information to support a row action without another API change

## Slice
1. derive a row-level primary action from `preferred_follow_up_action`
2. render it next to `Inspect` on `/admin/eval-reports`
3. keep create routed through existing `POST /api/v1/cases`
4. keep reuse routed through canonical `/admin/cases` handoff links
5. add runtime smoke coverage for the row-level reuse action

## Validation
- focused admin eval-report runtime smoke
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

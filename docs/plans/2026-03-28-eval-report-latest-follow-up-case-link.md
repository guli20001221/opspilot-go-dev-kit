# 2026-03-28 Eval Report Latest Follow-up Case Link

## Goal

Expose the freshest linked case directly on durable eval-report list items so `/admin/eval-reports` can hand off into operator follow-up without forcing a detail fetch first.

## Scope

- extend case follow-up summaries with `latest_follow_up_case_id`
- propagate the field through `GET /api/v1/eval-reports`
- render `Open latest case` on `/admin/eval-reports` list rows
- update OpenAPI, docs, and skill guidance

## Validation

- targeted case summary and eval-report HTTP tests
- `/admin/eval-reports` HTML/runtime smoke coverage
- `go test ./...`
- `tasks.ps1 check`

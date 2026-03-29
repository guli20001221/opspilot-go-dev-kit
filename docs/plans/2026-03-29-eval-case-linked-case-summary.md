# 2026-03-29 Eval Case Linked Case Summary

## Goal
Expose canonical linked-case pressure on eval-case reads so `/admin/evals` can render total/open/latest follow-up state without browser-side inference.

## Scope
- add `linked_case_summary` to eval-case list and detail responses
- render the summary in `/admin/evals` rows and detail
- update OpenAPI, docs, and skill guidance

## Validation
- targeted `go test` for `internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

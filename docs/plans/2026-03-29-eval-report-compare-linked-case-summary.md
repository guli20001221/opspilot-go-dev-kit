# 2026-03-29 Eval Report Compare Linked Case Summary

## Goal
Expose per-side linked follow-up pressure and ownership on the canonical eval-report compare contract so `/admin/eval-report-compare` does not infer queue state from only latest linked case IDs.

## Scope
- add `linked_case_summary` to each side of `GET /api/v1/eval-report-compare`
- render total/open/latest owner summary on compare cards
- update OpenAPI, docs, and skill guidance

## Validation
- targeted `go test` for compare endpoint and compare admin page
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

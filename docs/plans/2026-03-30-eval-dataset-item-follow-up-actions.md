# 2026-03-30 Eval Dataset Item Follow-Up Actions

## Goal
Expose member-level create-versus-reuse follow-up actions on canonical eval-dataset detail reads.

## Why
- `/admin/eval-datasets` already had member-level linked-case summary and linked-case handoff.
- The primary operator question remained unresolved for one dataset member: should they create a case, open an existing case, or open the canonical queue?
- This slice keeps that decision backend-owned and consistent with eval-case, eval-report, and bad-case lanes.

## Slice
- add `preferred_follow_up_action` to `GET /api/v1/eval-datasets/{dataset_id}` `items[]`
- wire `/admin/eval-datasets` member rows to consume that field for `Create case from item` versus reuse
- update OpenAPI, docs, and skill guidance

## Validation
- targeted `go test` for eval-dataset handlers and admin page HTML
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI parse validation

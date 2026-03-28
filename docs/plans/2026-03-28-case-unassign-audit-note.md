# 2026-03-28 Case Unassign Audit Note

## Goal
Make `POST /api/v1/cases/{case_id}/unassign` a canonical, auditable backend action instead of a pure state flip.

## Scope
- persist `unassigned_by` as an append-only case note
- keep `already unassigned` and `closed` cases on `409 invalid_case_state`
- keep memory and PostgreSQL stores behaviorally aligned
- update OpenAPI, docs, and admin skill guidance

## Verification
- targeted `go test` for `internal/case`, `internal/storage/postgres`, and `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

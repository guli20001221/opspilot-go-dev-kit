# 2026-03-28 Case Queue Row Assign

## Goal
Let operators claim compare-derived follow-up cases directly from the canonical queue without opening case detail first.

## Scope
- add a row-level `Assign to me` action for open unassigned case rows on `/admin/cases`
- reuse the canonical `POST /api/v1/cases/{case_id}/assign` endpoint
- refresh both the row and detail pane after assignment
- cover the action in admin cases runtime smoke

## Verification
- targeted admin cases page tests
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-28 Case Compare Provenance

## Goal
Persist structured eval-report comparison lineage on durable cases so compare-created follow-up work can hand operators back to the exact compare slice.

## Scope
- add canonical case fields for left and right eval report IDs plus the selected side
- persist and read those fields through the PostgreSQL and in-memory case stores
- expose `compare_origin` on `POST /api/v1/cases` and `GET /api/v1/cases/{case_id}`
- send `compare_origin` from `/admin/eval-report-compare` create-case actions
- render compare provenance plus `Open compare origin` on `/admin/cases`

## Verification
- targeted case service, PostgreSQL store, and HTTP API tests for compare provenance round-trip
- admin cases HTML/runtime smoke to confirm compare-origin rendering and handoff
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

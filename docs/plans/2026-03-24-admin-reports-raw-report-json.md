# 2026-03-24 admin reports raw report json

## Goal
Let operators inspect the durable report artifact directly on `/admin/reports` without adding a new backend debug contract.

## Decision
- Reuse the existing `GET /api/v1/reports/{report_id}` response already fetched by the report detail panel.
- Add `Show raw report JSON` and `Copy raw report JSON` to the existing report detail actions.
- Keep audit timeline and Temporal provenance on the existing task detail path.

## Validation
- report page HTML test covers the new actions
- targeted `go test` for admin page rendering
- full `go test ./...`
- browser smoke on `/admin/reports`

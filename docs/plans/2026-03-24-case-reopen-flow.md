# 2026-03-24 Case Reopen Flow

## Goal

Let operators recover from premature or mistaken case closure without creating a brand-new case.

## Scope

- add `POST /api/v1/cases/{case_id}/reopen`
- transition closed cases back to `open`
- clear `closed_by` on reopen
- append a durable operator note recording who reopened the case
- expose `Reopen case` only on closed case detail in `/admin/cases`
- after reopen, return the browser to the open queue

## Validation

- `go test ./internal/case ./internal/app/httpapi -count=1`
- `go test ./...`
- OpenAPI parse check
- browser smoke for close -> reopen on `/admin/cases`

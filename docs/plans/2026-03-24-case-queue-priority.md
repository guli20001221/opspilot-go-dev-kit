# 2026-03-24 Case Queue Priority

## Goal

Make `/admin/cases` behave more like an operator queue without changing the backend contract.

## Scope

- foreground `My open cases` and `Unassigned` as the main quick views
- add visible queue summary for owned and unassigned open cases in the current slice
- surface task-only versus report-backed provenance directly in the list and detail panes
- keep assign, close, notes, and existing case APIs unchanged

## Validation

- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
- browser smoke for `/admin/cases` quick-view switching and provenance rendering

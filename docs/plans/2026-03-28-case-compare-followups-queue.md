# 2026-03-28 Case Compare Follow-ups Queue

## Goal
Turn compare-origin provenance into a canonical operator queue by adding a durable case-list filter and a matching `/admin/cases` quick view.

## Scope
- add `compare_origin_only` to the canonical case list filter
- support that filter in memory and PostgreSQL case stores
- expose the filter on `GET /api/v1/cases`
- wire a `Compare follow-ups` quick view on `/admin/cases`
- add API, store, and admin smoke coverage for the new queue slice

## Verification
- targeted case list API and PostgreSQL filter tests
- admin cases HTML/runtime smoke for the quick view
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

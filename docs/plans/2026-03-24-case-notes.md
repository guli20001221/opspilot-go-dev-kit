# 2026-03-24 Case Notes

## Goal

Make `case` a minimally collaborative operator object by adding append-only notes that are durable, tenant-scoped, and visible in `/admin/cases`.

## Scope

- add durable `case_notes` storage
- add `POST /api/v1/cases/{case_id}/notes`
- expose recent notes in `GET /api/v1/cases/{case_id}`
- render notes and note creation in `/admin/cases`
- include the latest note in copied case handoff summaries

## Non-goals

- threaded comments
- rich text
- mentions or notifications
- search
- reopen/sub-status workflow expansion

## Validation

- targeted `go test` for `internal/case`, `internal/app/httpapi`, and `internal/storage/postgres`
- `go test ./...`
- OpenAPI parse validation
- browser smoke on `/admin/cases`

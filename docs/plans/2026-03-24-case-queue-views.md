# 2026-03-24 Case Queue Views

## Goal

Turn `/admin/cases` from a generic case browser into a minimally operable queue by surfacing open owned work first.

## Scope

- add `assigned_to` filtering to `GET /api/v1/cases`
- default `/admin/cases` to `status=open`
- add `Open cases` and `My open cases` quick views
- show age/staleness from canonical `updated_at`

## Non-goals

- team queues
- SLA timers
- identity-bound authorization semantics for `me`
- extra lifecycle states

## Validation

- targeted tests for case list filtering
- `go test ./...`
- OpenAPI parse validation
- `/admin/cases` browser smoke for queue views

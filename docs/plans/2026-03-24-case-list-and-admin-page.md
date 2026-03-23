# 2026-03-24 case list and admin page

## Goal

Promote durable cases from single-record storage into a usable operator slice.

## Scope

- add `GET /api/v1/cases` with tenant, status, source-task, source-report, limit, and offset filters
- add case list support to the in-memory and PostgreSQL case stores
- add `/admin/cases` as a minimal embedded operator page
- keep the page read-only and reuse existing task/report endpoints for handoff

## Notes

- the page intentionally does not add case actions yet
- task and report drill-down stay rooted in their canonical endpoints

# 2026-03-23 Admin Taskboard Endpoint

## Goal

Expose the first admin-facing task board contract so future `web/admin` pages can consume a backend read model instead of recomputing task summaries in the browser.

## Scope

- add `GET /api/v1/admin/task-board`
- reuse the existing task list filters
- return task items, page metadata, and visible-slice summary counts
- keep the endpoint read-only and thin over `internal/app/admin/taskboard`
- update OpenAPI, architecture notes, runbook, README, and admin skill guidance

## Key decisions

- no new write actions or drill-down fields in this slice
- reuse `TaskSummary` for board items so task-list and admin-board rows stay aligned
- keep summary counts scoped to the visible filtered page for now; no whole-dataset aggregation query yet

## Validation

- targeted HTTP handler tests for the new admin endpoint
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI parsing

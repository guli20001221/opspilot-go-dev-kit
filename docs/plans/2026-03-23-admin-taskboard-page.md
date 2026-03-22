# 2026-03-23 Admin Taskboard Page

## Goal

Add the first operator-facing `web/admin` page without introducing a separate frontend toolchain or moving task-board logic into the browser.

## Scope

- serve an embedded `GET /admin/task-board` page from the API process
- render summary cards, filter controls, and task rows from `GET /api/v1/admin/task-board`
- keep the page read-only
- include loading, empty, and error states
- update README, architecture notes, runbook, and admin skill guidance

## Key decisions

- use an embedded static HTML page under `web/admin` so the first admin UI slice stays easy to run in the existing Go stack
- reuse the backend read model and current filter contract instead of recomputing summaries in browser code
- keep pagination simple by driving `limit` and `offset` directly from the page URL and controls

## Validation

- targeted HTTP tests for the page route and existing admin endpoint
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- compose-level smoke test that loads `/admin/task-board` against real task data

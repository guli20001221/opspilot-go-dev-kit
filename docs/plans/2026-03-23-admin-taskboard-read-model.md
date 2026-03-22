# 2026-03-23 Admin Taskboard Read Model

## Goal

Add the first backend read model for future `web/admin` task views so the frontend does not need to recompute task-state summaries or pagination metadata on its own.

## Scope

- add `internal/app/admin/taskboard`
- consume existing `workflow.Service.ListTasks` output
- return operator-facing task rows plus visible-slice summary counts
- preserve pagination metadata from the workflow task page
- keep the package backend-only for now; no new public API endpoints

## Key decisions

- keep the first slice narrow: summarize only the currently visible page, not the entire filtered dataset
- do not expose a new HTTP contract yet; this package is a stable backend seam for upcoming `web/admin` work
- keep summary fields typed and explicit instead of returning generic maps

## Validation

- targeted `go test ./internal/app/admin/taskboard`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

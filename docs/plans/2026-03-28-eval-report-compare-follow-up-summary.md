# 2026-03-28 Eval Report Compare Follow-up Summary

## Goal

Expose per-side follow-up pressure on `/admin/eval-report-compare` so operators can decide whether new case creation is needed without leaving the compare surface.

## Scope

- extend compare items with follow-up summary fields already available from canonical case lineage
- render those summary fields next to the latest-case handoff on both compare cards
- add focused API and admin runtime coverage
- sync OpenAPI, docs, and admin-console guidance

## Validation

- targeted compare API and runtime smoke tests
- `go test ./...`
- `tasks.ps1 check`

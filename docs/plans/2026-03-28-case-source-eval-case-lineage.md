# 2026-03-28 Case Source Eval Case Lineage

## Goal

Land precise bad-case follow-up on top of the existing durable case contract without inventing a second case surface.

## Scope

- add `source_eval_case_id` to durable cases
- validate that a selected eval case belongs to the selected eval report
- reuse open follow-up by `tenant_id + source_eval_case_id`
- keep report-level reuse only for requests without `source_eval_case_id`
- add `/admin/eval-reports` row-level `Create case from bad case`
- surface source eval-case lineage on `/admin/cases`

## Why

Report-level follow-up is too coarse once operators need to track one failing eval case separately from the broader eval report.

## Validation

- targeted `go test` for `internal/case`, `internal/storage/postgres`, and `internal/app/httpapi`
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI YAML parse


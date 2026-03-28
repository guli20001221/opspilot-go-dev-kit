# 2026-03-28 eval-report-compare follow-up queue summary

## Goal

Expose compare-origin follow-up pressure as backend-owned per-side summary on the canonical eval-report compare contract, then hand operators into the existing case queue instead of rebuilding compare lineage in the browser.

## Scope

- extend `GET /api/v1/eval-report-compare` with compare-derived follow-up summary per side
- surface compare-origin queue handoff links on `/admin/eval-report-compare`
- add typed coverage for compare summary fields and page runtime handoff behavior
- sync OpenAPI, runbooks, and skill guidance

## Notes

- compare-derived queue summary is narrower than general eval-report follow-up summary
- the queue handoff reuses `/admin/cases?source_eval_report_id=...&compare_origin_only=true&status=open`
- unresolved bad-case handoff remains separate and continues to target the eval-report lane

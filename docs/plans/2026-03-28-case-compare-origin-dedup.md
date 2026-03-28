# 2026-03-28 case compare-origin dedup

## Goal

Make compare-driven follow-up case creation canonical on the backend so repeated handoff for the same report side of the same comparison reuses the existing open case.

## Scope

- add exact compare-origin lookup to case storage
- reuse an open case on `POST /api/v1/cases` when `tenant_id + source_eval_report_id + compare_origin(left/right/selected_side)` already exists
- preserve distinct case creation for different compare lineage
- sync API docs and operator runbooks

## Notes

- plain eval-report follow-up dedupe remains separate from compare-origin dedupe
- compare-origin dedupe is exact-match only; different left/right lineage must remain distinct

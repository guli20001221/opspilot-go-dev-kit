# 2026-03-28 Case Source Eval Report Dedupe

## Goal
Make eval-report follow-up creation canonical so repeated `Create case` actions do not generate duplicate open regression cases for the same tenant and source eval report.

## Scope
- reuse the newest open case for `tenant_id + source_eval_report_id` when `compare_origin` is absent
- preserve compare-origin case creation as distinct follow-up work
- update API tests, admin runtime smoke, OpenAPI, and operator docs

## Non-goals
- deduping compare-origin follow-up cases
- changing task- or report-backed case creation semantics
- adding a new admin-only write endpoint

## Acceptance
- `POST /api/v1/cases` returns `200` plus the existing case when the same eval-report follow-up is already open
- repeated `Create case` clicks from `/admin/eval-reports` deep-link to the canonical open case
- compare-origin case creation still returns `201` and new case IDs

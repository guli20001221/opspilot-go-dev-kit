# 2026-03-28 Eval-Case Direct Case Creation

## Goal

Let operators create or reuse precise follow-up directly from `/admin/evals` by using standalone `source_eval_case_id` on the canonical case contract.

## Scope

- allow `POST /api/v1/cases` with `source_eval_case_id` and no `source_eval_report_id`
- keep tenant validation on the canonical eval-case read
- preserve open-case idempotent reuse for the same `tenant_id + source_eval_case_id`
- add `Create case` on `/admin/evals`

## Notes

- do not add a new endpoint
- do not require the operator to detour through `/admin/eval-reports`
- keep compare-origin behavior distinct from plain eval-case follow-up

# 2026-03-28 Eval Report Linked Case Summary

## Goal
Move linked-case summary on `/admin/eval-reports` into the canonical eval-report detail contract.

## Why
- the page was still issuing a second `/api/v1/cases?source_eval_report_id=...` request after detail load
- the linked-case card only needs stable summary data: counts, latest status, and latest assignee
- that summary belongs on the report detail read model, not in browser-side derivation

## Slice
1. add `linked_case_summary` to `GET /api/v1/eval-reports/{report_id}`
2. populate it from canonical case follow-up summary plus the latest linked case owner
3. remove `loadLinkedCases()` from `/admin/eval-reports`
4. update tests, OpenAPI, README, architecture, and admin skill guidance

## Validation
- focused `go test` on eval-report contract and admin page smoke
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-28 Eval-Report Compare Follow-Up Actions

## Goal

Move the left/right compare-origin follow-up decision for `/admin/eval-report-compare` out of browser heuristics and into the canonical compare contract.

## Scope

- add typed `preferred_compare_follow_up_action` to each side of `GET /api/v1/eval-report-compare`
- keep the field backend-owned and derived from canonical compare follow-up summary already present on the compare contract
- switch the left/right primary buttons on `/admin/eval-report-compare` to consume the field
- keep existing compare queue links and case creation payloads unchanged
- update OpenAPI, README, architecture docs, runbook, and admin skill guidance

## Notes

- this slice does not add a new endpoint
- this slice does not change compare-origin case deduplication behavior
- the page keeps a small fallback heuristic only for compatibility with older payloads during rollout

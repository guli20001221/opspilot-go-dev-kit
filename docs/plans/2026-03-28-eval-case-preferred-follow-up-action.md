# 2026-03-28 Eval-Case Preferred Follow-Up Action

## Goal

Move the create-versus-reuse follow-up decision for eval cases out of `/admin/evals` browser heuristics and into the canonical eval-case read contract.

## Scope

- add typed `preferred_follow_up_action` to `GET /api/v1/eval-cases` rows and `GET /api/v1/eval-cases/{eval_case_id}`
- keep the field backend-owned and derived from canonical follow-up summary already present on eval cases
- switch `/admin/evals` to consume that field for primary action rendering and row-level reuse handoff
- update OpenAPI, README, architecture docs, runbook, and skill guidance

## Notes

- this slice does not change case storage or deduplication behavior
- this slice does not add new endpoints
- fallback browser heuristics remain only as compatibility for older payloads during rollout

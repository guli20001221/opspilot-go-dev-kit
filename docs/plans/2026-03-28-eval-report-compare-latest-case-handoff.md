# 2026-03-28 Eval Report Compare Latest Case Handoff

## Goal

Expose canonical latest-case handoff for both sides of `/admin/eval-report-compare` so operators can inspect existing follow-up before creating a new case.

## Scope

- extend `GET /api/v1/eval-report-compare` with `latest_follow_up_case_id` on left/right items
- render `Open left latest case` and `Open right latest case` in the compare UI
- add compare API and runtime smoke coverage
- update OpenAPI, docs, and admin-console guidance

## Validation

- targeted compare API and admin runtime tests
- `go test ./...`
- `tasks.ps1 check`

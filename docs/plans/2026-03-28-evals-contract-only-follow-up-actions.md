# 2026-03-28 Evals Contract-Only Follow-Up Actions

## Goal
Remove browser-side count heuristics from `/admin/evals` now that canonical eval-case reads already expose `preferred_follow_up_action`.

## Why
- eval-case list and detail already carry a backend-owned create-versus-reuse decision
- the page still had fallback logic based on `open_follow_up_case_count` and `latest_follow_up_case_id`
- keeping both paths alive risks UI behavior drifting from the canonical API

## Slice
1. remove count-based fallback from `/admin/evals` `preferredFollowUpAction(item)`
2. leave the safe default as `create` when the typed field is absent
3. rely on existing runtime smoke to prove reuse still works when the canonical field is present

## Validation
- `go test ./internal/app/httpapi -run 'TestAdminEvalsPageRendersHTML|TestAdminEvalsPageRuntimeSmoke' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

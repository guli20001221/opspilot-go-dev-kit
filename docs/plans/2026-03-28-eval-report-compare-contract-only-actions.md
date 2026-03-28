# 2026-03-28 Eval Report Compare Contract-Only Actions

## Goal
Remove browser-side compare-follow-up heuristics from `/admin/eval-report-compare` once each side already exposes `preferred_compare_follow_up_action`.

## Why
- compare left/right actions already have a backend-owned create-versus-queue decision
- the page still had a fallback based on `open_compare_follow_up_case_count`
- keeping both paths alive risks operator behavior drifting from the canonical compare contract

## Slice
1. remove count-based fallback from compare `sidePrimaryAction(side)`
2. keep the safe default as `create` when the typed field is absent
3. rely on existing compare runtime smoke to prove queue reuse still works when the canonical field is present

## Validation
- `go test ./internal/app/httpapi -run 'TestCompareEvalReportsReturnsTypedSummary|TestAdminEvalReportComparePageRendersHTML|TestAdminEvalReportComparePageRuntimeSmoke' -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-29 Eval Dataset Recent-Run Case Handoff

## Goal
Carry run-backed follow-up context directly in `recent_runs[]` on eval-dataset detail so `/admin/eval-datasets` can hand operators into the latest run-backed case or queue without detouring through `/admin/eval-runs`.

## Scope
- add linked case summary fields to each recent run row
- add a typed preferred case action for each recent run row
- render recent-run case handoff in the dataset detail page

## Guardrails
- no new endpoints
- no browser-side case reuse heuristics
- preserve existing dataset-level and latest-report-level case handoff behavior

## Verification
- targeted `go test` for eval-dataset HTTP and admin page smoke coverage
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

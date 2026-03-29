# 2026-03-29 eval-dataset preferred run-backed case action

## Goal
Keep `/admin/eval-datasets` contract-first when operators triage latest-run follow-up work.

## Slice
- add `preferred_run_backed_case_action` to canonical eval-dataset list/detail reads
- reuse durable `source_eval_run_id` case lineage
- support both `open_existing_case` and `open_existing_queue`
- keep browser code free of run-backed case routing heuristics

## Validation
- targeted `go test` for eval-dataset API and admin page runtime smoke
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

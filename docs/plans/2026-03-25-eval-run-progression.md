# 2026-03-25 Eval Run Progression

## Goal

Turn durable eval-run kickoff records into the first executable eval lifecycle.

## Scope

- Add claim/update primitives for eval runs in memory and PostgreSQL stores.
- Add `internal/eval.Runner`.
- Wire the worker to process queued eval runs.
- Advance runs through `queued -> running -> succeeded|failed`.
- Persist `started_at`, `finished_at`, and `error_reason`.
- Expose those fields through the existing run detail endpoint and `/admin/eval-runs`.

## Deliberate non-goals

- No judge prompts or model execution yet.
- No Temporal orchestration for eval runs yet.
- No score aggregation, report artifacts, or compare UI yet.
- No retry surface yet.

## Decisions

- Reuse the canonical `eval_runs` record as the only status surface.
- Keep the first execution body as a placeholder worker executor.
- Add one dev-only fault-injection switch, `OPSPILOT_EVAL_RUN_FAIL_ALL`, for operator-path validation.
- Keep `/admin/eval-runs` contract-first by reading only the canonical run API.

## Validation

- `go test ./internal/eval ./internal/app/config ./internal/app/httpapi ./internal/storage/postgres -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

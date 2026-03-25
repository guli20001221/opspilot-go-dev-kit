# 2026-03-25 Eval Run Kickoff

## Goal

Land the first durable eval-run contract so operators can kick off regression work from a published dataset before judge execution is wired.

## Scope

- Add a durable `eval_runs` table and PostgreSQL store.
- Add `RunService` in `internal/eval`.
- Add `POST /api/v1/eval-runs`.
- Add `GET /api/v1/eval-runs`.
- Add `GET /api/v1/eval-runs/{run_id}`.
- Add `/admin/eval-runs`.
- Add `Run dataset` handoff from `/admin/eval-datasets`.

## Deliberate non-goals

- No worker-side eval execution yet.
- No Temporal orchestration yet for eval runs.
- No judge prompts, score aggregation, or eval report generation yet.
- No compare UI for eval runs yet.

## Decisions

- Eval runs snapshot `dataset_id`, `dataset_name`, and `dataset_item_count` at kickoff time so later execution can stay reproducible.
- Only published datasets can start a run.
- The first run lifecycle stops at durable queued records; execution wiring will come later.
- The first admin run lane reuses canonical run list/detail contracts and existing dataset/eval handoff links instead of inventing admin-only state.

## Validation

- `go test ./internal/eval ./internal/app/httpapi ./internal/storage/postgres -count=1`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- `python -c "import pathlib, yaml; yaml.safe_load(pathlib.Path('docs/openapi/openapi.yaml').read_text())"`

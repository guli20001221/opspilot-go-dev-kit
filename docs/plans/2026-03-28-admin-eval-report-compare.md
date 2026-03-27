# 2026-03-28 Admin Eval Report Compare

## Goal

Land the first durable eval-report comparison slice without inventing a browser-owned diff model.

## Scope

- add a canonical read-only compare contract:
  - `GET /api/v1/eval-report-compare`
- derive compare summary in `internal/eval`
- add `/admin/eval-report-compare`
- keep handoff paths to eval reports, eval runs, and version detail

## Key decisions

- mirror the existing runtime report-compare pattern instead of creating a new compare architecture
- require `tenant_id` at the compare boundary so cross-tenant reads stay impossible
- keep compare summary narrow:
  - dataset alignment
  - run-status alignment
  - judge-version drift
  - metadata drift
  - top-line metric deltas
  - bad-case overlap
  - ready-at delta
- reuse the full eval-report detail contract for left/right panes instead of inventing compare-only report snapshots

## Validation

- targeted `go test` for `internal/eval`
- targeted `go test` for `internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- OpenAPI YAML parse validation

## Follow-up

- promote `/admin/eval-reports` detail handoff into a prefilled compare flow
- consider a runtime smoke test for `/admin/eval-report-compare` once browser binaries are reliably available in test environments
- next likely vertical slice: durable eval-report compare deltas on bad-case membership and direct handoff into shared trace/version surfaces

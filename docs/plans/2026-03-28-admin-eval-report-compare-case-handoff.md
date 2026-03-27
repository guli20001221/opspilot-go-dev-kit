# 2026-03-28 Admin Eval Report Compare Case Handoff

## Goal

Turn eval-report comparison from a read-only inspection lane into a durable operator handoff.

## Scope

- add `Create case` on `/admin/eval-report-compare`
- reuse canonical `POST /api/v1/cases`
- deep-link successful creates into `/admin/cases`
- keep compare page read-only apart from this handoff action

## Key decisions

- do not add an admin-only case creation endpoint
- use the right-side report as `source_report_id` for the created case
- summarize the comparison into a compact case summary instead of inventing compare-specific backend write contracts
- keep failure reporting explicit in the page status note

## Validation

- targeted `go test` for `internal/app/httpapi`
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

## Follow-up

- prefill compare from `/admin/eval-reports` detail without hand-entering both report IDs
- consider a future backend helper if compare-to-case summary formatting needs to be shared across multiple surfaces

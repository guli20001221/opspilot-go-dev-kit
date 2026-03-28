# 2026-03-28 Case Queue Row Close

## Goal
Let operators resolve compare-derived follow-up cases directly from the canonical queue without opening case detail first.

## Scope
- add a row-level `Close from queue` action for open case rows on `/admin/cases`
- reuse the canonical `POST /api/v1/cases/{case_id}/close` endpoint
- refresh queue state after close so the case leaves the open compare slice immediately
- cover the action in admin cases runtime smoke

## Verification
- targeted admin cases page tests
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

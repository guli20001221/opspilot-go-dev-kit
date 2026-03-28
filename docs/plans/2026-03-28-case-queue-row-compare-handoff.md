# 2026-03-28 Case Queue Row Compare Handoff

## Goal
Let operators jump from a compare-derived case row straight back into the exact eval-report comparison without opening case detail first.

## Scope
- add a row-level `Open compare` action on `/admin/cases` for rows with compare provenance
- derive the link from canonical case list data already on the row payload
- cover the new handoff in static HTML assertions and runtime smoke

## Verification
- targeted admin cases page tests
- full `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

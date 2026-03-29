## Goal

Move `/admin/eval-report-compare` linked-case handoff off browser-side `latest_follow_up_case_id` heuristics and onto a canonical backend-owned action field.

## Scope

- add `preferred_linked_case_action` to each `EvalReportComparisonItem`
- keep `linked_case_summary` as the ownership and pressure summary block
- update `/admin/eval-report-compare` to consume the typed action for left and right linked-case handoff
- extend OpenAPI and compare-focused tests

## Why

The compare page already had backend-owned `preferred_compare_follow_up_action`, but linked-case handoff still branched in the browser on `latest_follow_up_case_id`. That let UI heuristics drift from canonical queue semantics when the latest linked case was closed.

## Validation

- targeted compare contract and page tests
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

# 2026-03-28 Eval Report Detail Latest Case Handoff

## Goal

Make the freshest follow-up case handoff visible from the selected eval-report detail pane, not only from the table row.

## Scope

- render `Open latest case` in `/admin/eval-reports` detail when `latest_follow_up_case_id` exists
- keep the existing row-level handoff unchanged
- add runtime coverage for the detail-pane CTA
- sync docs and admin-console guidance

## Validation

- targeted `/admin/eval-reports` HTML and runtime smoke tests
- `go test ./...`
- `tasks.ps1 check`

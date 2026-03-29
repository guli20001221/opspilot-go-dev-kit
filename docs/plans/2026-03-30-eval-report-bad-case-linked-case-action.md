## Goal

Move `/admin/eval-reports` bad-case linked-case handoff off browser-side `latest_follow_up_case_id` heuristics and onto a canonical backend-owned action field.

## Scope

- add `preferred_linked_case_action` to each `bad_cases[]` row on `GET /api/v1/eval-reports/{report_id}`
- keep `preferred_follow_up_action` for backward compatibility
- update `/admin/eval-reports` bad-case rows to consume the new typed action
- add queue-fallback coverage when the latest bad-case linked case is closed

## Why

The report detail already had typed follow-up actions, but bad-case linked-case handoff still branched in the browser on `latest_follow_up_case_id`. That produced the wrong operator action when historical follow-up existed but the latest bad-case case was closed.

## Validation

- targeted eval-report detail and admin page tests
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`

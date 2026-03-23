# 2026-03-23 Admin Taskboard Row Sync

## Goal

Preserve table context while operators navigate task details by keeping the selected task row highlighted and synced with the current detail selection.

## Scope

- add a visible selected-row treatment to the embedded task board table
- sync the selected-row state with explicit task inspection and previous/next detail navigation
- mark the selected row with `aria-current` for clearer DOM state
- keep the behavior entirely frontend-local without changing backend contracts
- update docs and admin skill guidance

## Key decisions

- do not add a second selection model beyond the existing `task_id` URL state
- use the table row as the single visual anchor for list context
- scroll the selected row back into the nearest visible area when detail navigation changes the current task

## Validation

- failing-then-passing page test for selected-row affordance
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke showing selected-row highlight moving with detail navigation

# 2026-03-23 Admin Taskboard Lane Focus

## Goal

Let operators narrow the current board to tasks similar to the selected task without manually copying values back into the filter form.

## Scope

- add a `Focus same lane` control to the embedded task detail panel
- write the selected task's `tenant_id`, `task_type`, `reason`, and `requires_approval` back into the existing board filters
- reset `status` and `offset` so the board shows the whole lane instead of just the current status bucket
- keep using the same `/api/v1/admin/task-board` backend contract and URL query state
- update docs and admin skill guidance

## Key decisions

- do not add backend "similar tasks" endpoints
- keep the action transparent by mutating the visible filter form and URL
- treat lane focus as an operator convenience on top of the existing board model, not a new query state machine

## Validation

- failing-then-passing page test for the lane focus affordance
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke showing the board shrinking to the selected task lane after clicking `Focus same lane`

# 2026-03-23 Admin Taskboard Status Focus

## Goal

Let operators pivot from a selected task into the broader status queue without manually setting the board status filter.

## Scope

- add a `Focus same status` control to the embedded task detail panel
- write the selected task `status` back into the existing board filter form while keeping the current tenant scope
- reset `offset` so the resulting queue starts from the first page
- keep the behavior entirely within the existing `/api/v1/admin/task-board` contract and URL query model
- update docs and admin skill guidance

## Key decisions

- do not add a separate queue endpoint or backend status summary
- keep the action explicit by mutating the existing filter controls and URL state
- allow the rest of the filter model to remain visible so operators can refine further after the pivot

## Validation

- failing-then-passing page test for the status focus affordance
- `go test ./...`
- `powershell -ExecutionPolicy Bypass -File scripts/dev/tasks.ps1 check`
- browser smoke showing the board narrowing to the selected status after clicking `Focus same status`

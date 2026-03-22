# 2026-03-22 Approved Tool Fault Injection

## Scope

Add a development-only fault-injection path for `approved_tool_execution` so the Temporal failure and retry flow can be exercised end-to-end without changing the public task API.

## Change

- add `OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE` to worker configuration
- pass the current approve or retry action into the approved-tool activity
- fail the activity only for the `approve` action when fault injection is enabled
- leave `retry` successful so operator recovery can be verified

## Expected outcome

- local environments can prove that `approve -> failed -> retry -> succeeded` works
- the same stable `task_id` can be reused while each retry receives a fresh Temporal run

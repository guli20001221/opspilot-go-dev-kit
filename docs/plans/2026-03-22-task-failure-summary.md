# 2026-03-22 Task Failure Summary

## Scope

Normalize task failure reasons so operators see a short root-cause summary instead of the full wrapped Temporal error chain.

## Change

- summarize execution errors in the worker failure path before persisting `error_reason`
- keep the external task API unchanged
- cover the new behavior in workflow and HTTP tests

## Expected outcome

- `GET /api/v1/tasks/{task_id}` returns a readable `error_reason`
- worker logs still hold the deeper Temporal error context for debugging

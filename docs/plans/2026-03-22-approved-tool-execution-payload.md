# 2026-03-22 Approved Tool Execution Payload

## Scope

Move `approved_tool_execution` from a pure placeholder activity to a real registered-tool execution path when the task was promoted from chat.

## Change

- add internal tool payload fields to workflow tasks
- persist tool name and arguments on approval-required promotions
- let the Temporal approved-tool activity execute the registered tool with `ApprovalGranted=true`
- keep a compatibility fallback for legacy approval tasks that do not have payload

## Expected outcome

- chat-promoted approval tasks carry the exact tool the worker should run after approval
- approved execution no longer depends on reconstructing intent from generic task type alone
- existing manually created approval tasks remain runnable during the migration period

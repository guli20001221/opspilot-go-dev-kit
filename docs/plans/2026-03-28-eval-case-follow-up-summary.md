# 2026-03-28 Eval-Case Follow-Up Summary

## Goal

Expose canonical follow-up case summary on eval-case reads so `/admin/evals` can hand operators into durable case work without extra browser-side case queries.

## Scope

- add follow-up summary fields to `GET /api/v1/eval-cases`
- add follow-up summary fields to `GET /api/v1/eval-cases/{eval_case_id}`
- derive the summary from canonical case storage through `source_eval_case_id`
- update `/admin/evals` to show latest follow-up handoff and full follow-up-slice handoff

## Notes

- keep the eval-case contract lightweight: counts plus latest linked case only
- keep follow-up details canonical on `/api/v1/cases` instead of duplicating case payloads into eval storage
- prefer handoff through `latest_follow_up_case_id` or `/admin/cases?source_eval_case_id=...`

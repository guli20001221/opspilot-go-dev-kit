# 2026-03-24 case close action

## Goal

Add the smallest durable lifecycle transition for operator cases so a follow-up object can move from `open` to `closed` without inventing a parallel admin-only mutation path.

## Scope

- add `closed` as a first-class case status
- persist `closed_by` on case records
- expose `POST /api/v1/cases/{case_id}/close`
- wire `/admin/cases` to the new canonical close endpoint
- update OpenAPI, runbook, README, architecture notes, and admin skill guidance

## Notes

- tenant scoping remains fail-closed on detail and close actions
- case list/status filters continue to reuse the canonical `status` field rather than adding a separate lifecycle view
- the admin page stays thin and derives UI state from the existing case list and case detail contracts

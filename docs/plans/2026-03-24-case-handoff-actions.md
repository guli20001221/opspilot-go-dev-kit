# 2026-03-24 case handoff actions

## Goal

Make `/admin/cases` useful for operator handoff without expanding the backend contract.

## Scope

- add `Copy case summary`
- add `Copy case link`
- add `Open case API detail`
- keep all data derived from `GET /api/v1/cases/{case_id}`

## Notes

- no new backend endpoint or formatter
- the page remains contract-first and reuses the canonical case JSON for debugging and escalation

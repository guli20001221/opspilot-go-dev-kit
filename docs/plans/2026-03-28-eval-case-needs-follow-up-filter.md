# 2026-03-28 Eval-Case Needs-Follow-Up Filter

## Goal

Turn eval-case follow-up summary into a real operator queue by adding a canonical `needs_follow_up` filter and a matching `/admin/evals` quick view.

## Scope

- add `needs_follow_up=true|false` to `GET /api/v1/eval-cases`
- keep filtering backend-owned rather than browser-derived
- add a `Needs follow-up` quick view on `/admin/evals`
- reuse the existing eval-case summary and follow-up handoff links

## Notes

- do not add a new endpoint
- do not duplicate case state into eval storage
- keep `/admin/evals` on the canonical eval-case contract

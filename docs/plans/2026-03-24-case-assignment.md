# 2026-03-24 case assignment

## Goal

Make durable cases operationally actionable by adding explicit ownership.

## Scope

- add `assigned_to` and `assigned_at` to the case model
- expose `POST /api/v1/cases/{case_id}/assign`
- surface assignee state on `/admin/cases`
- keep the UI thin and reuse the canonical case contract

## Notes

- assignment only applies to open cases
- assignment is free-text in this first slice; auth-backed identity can tighten later
- no new admin-only write surface is introduced

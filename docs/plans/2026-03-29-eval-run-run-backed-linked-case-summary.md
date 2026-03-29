# 2026-03-29 Eval-Run Run-Backed Linked Case Summary

## Goal
Make `/api/v1/eval-runs` and `/api/v1/eval-runs/{run_id}` resolve `linked_case_summary` and `preferred_linked_case_action` from durable `source_eval_run_id` lineage, not from per-item eval-case follow-up inference.

## Why
- run-backed follow-up is already a canonical case lineage path
- eval-run operator pages should surface that queue even when no item-specific case exists
- frontend should consume one backend-owned summary instead of reconstructing run linkage from eval-case summaries

## Slice
- add `SummarizeBySourceEvalRunIDs` to the canonical case store/service
- implement run-backed case summary in memory and PostgreSQL stores
- switch eval-run read assembly to use run-backed linked-case summary
- add HTTP regression coverage for run-backed-only follow-up

## Non-goals
- no new API fields
- no new admin pages
- no change to per-item follow-up actions

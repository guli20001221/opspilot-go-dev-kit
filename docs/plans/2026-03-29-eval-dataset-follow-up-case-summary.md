# 2026-03-29 Eval Dataset Follow-up Case Summary

## Goal
Expose a canonical follow-up case pressure summary on `GET /api/v1/eval-datasets/{dataset_id}` so `/admin/eval-datasets` can show linked-case totals and latest case status before handing operators into `/admin/cases`.

## Scope
- add a detail-only `follow_up_case_summary` block to the eval-dataset response
- derive the block from existing latest-report case summaries
- render the summary in the dataset detail pane
- keep list rows and existing queue handoff fields backward compatible

## Notes
- do not introduce a new dataset-only case API
- do not move case aggregation into browser code
- keep `preferred_case_queue_action` authoritative for queue handoff

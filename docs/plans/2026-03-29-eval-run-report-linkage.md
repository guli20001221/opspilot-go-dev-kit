## Goal

Expose canonical run-to-report linkage on eval-run reads so `/admin/eval-runs` can hand operators directly into durable eval-report review.

## Scope

- add `report_id` and `report_status` to terminal `EvalRun` responses when the durable eval report exists
- keep queued and running rows lightweight and omit report linkage
- wire `/admin/eval-runs` list rows and detail handoff to `/admin/eval-reports` and the eval-report API
- update OpenAPI, docs, and admin skill guidance

## Notes

- use actual eval-report lookups from the HTTP read-model layer instead of browser-side ID derivation
- do not force the eval-run contract to carry full eval-report payloads; this slice is only about stable linkage

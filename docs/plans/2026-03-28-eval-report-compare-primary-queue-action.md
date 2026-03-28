# 2026-03-28 eval-report-compare primary queue action

## Goal

Reduce accidental duplicate compare-derived cases by turning the compare page's primary side action into a queue handoff whenever that side already has open compare-origin follow-up.

## Scope

- reuse existing compare queue summary fields from `GET /api/v1/eval-report-compare`
- switch the compare page button text and click behavior from `Create case` to `Open ... compare queue` when open compare-origin follow-up already exists
- keep the create path unchanged for sides that still have no open compare queue
- add runtime coverage for both queue-handoff and create-case paths

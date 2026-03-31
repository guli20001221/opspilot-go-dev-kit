## 2026-03-31 Eval Dataset Latest Activity Action

Goal:
- move dataset-level latest run/report handoff behind one backend-owned contract

Scope:
- add `preferred_latest_activity_action` to eval dataset list/detail
- use it in `/admin/eval-datasets` row/detail handoff
- keep recent run activity and case queue contracts unchanged

Why:
- remove remaining hardcoded latest-run versus latest-report routing from the dataset page
- keep dataset operator routing aligned with the same contract-first pattern already used in eval runs, reports, and compare views

# AGENTS.override.md

Local scope: `internal/workflow/**`

Priorities here:
- Temporal workflows orchestrate
- activities perform external I/O
- approval pauses and resumes are explicit
- every long-running job exposes status and retryability

Extra rules:
- no direct network or DB side effects inside workflow definitions
- activities must be idempotent or explicitly guarded
- surface operator-friendly failure reasons
- add workflow tests for branching, retry, and approval paths

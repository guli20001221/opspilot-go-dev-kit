# AGENTS.override.md

Local scope: `web/admin/**`

Priorities here:
- contract-first UI development
- backend remains source of truth
- task/case/report/version flows stay understandable
- trace deep links and reproducibility metadata are first-class

Extra rules:
- no business logic that should live in backend services
- do not invent fields not present in backend contracts
- expose report filters, case drill-down, and comparison views clearly
- prefer simple components over framework cleverness

# Claude hooks (recommended later)

Suggested hooks for this repository:

- `PreToolUse`
  - block edits to protected files without confirmation
  - warn before destructive migrations or write-capable tool flows
- `PostToolUse`
  - run `gofmt` or `goimports` on touched Go files
  - run focused lint or tests for the changed package
- `Notification` or `Stop`
  - remind to update ADRs, runbooks, or eval baselines when needed

Hooks are best for deterministic enforcement.
Keep judgment-heavy review in skills or subagents instead.

# AGENTS.override.md

Local scope: `internal/agent/**`

Priorities here:
- keep planning and orchestration typed and explicit
- keep provider SDK details behind internal interfaces
- keep prompts versioned outside handlers
- keep planner, tool, and critic logic independently testable

Extra rules:
- do not put orchestration state in free-form prompt text only
- do not pass opaque tool blobs between planner and critic
- prefer narrow packages with small interfaces
- add or update regression cases when routing or prompt behavior changes

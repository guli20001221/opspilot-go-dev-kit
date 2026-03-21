# Architecture summary

OpsPilot-Go is a production-oriented Golang multi-agent platform with these major layers:

- API / gateway
- application services under `internal/app/*`
- context engine
- Planner / Retrieval / Tool / Critic runtime
- workflow and approval layer
- retrieval and storage
- eval and observability
- admin console

The current foundation slice also includes a local development stack:

- PostgreSQL for application data and migrations
- Redis for future coordination and caching flows
- Temporal plus Temporal UI for workflow development
- API and worker processes bootstrapped through the same local Compose topology

The current Milestone 1 slice adds:

- `internal/session` for in-memory session and message persistence
- `internal/app/chat` as the application boundary for the synchronous chat skeleton
- `internal/app/httpapi` as a thin transport layer over the session and chat services

This file is intentionally brief in the AI development kit.
Promote it to the main repository and expand it as implementation begins.

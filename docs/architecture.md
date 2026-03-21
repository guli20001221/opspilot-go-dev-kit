# Architecture summary

OpsPilot-Go is a production-oriented Golang multi-agent platform with these major layers:

- API / gateway
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

This file is intentionally brief in the AI development kit.
Promote it to the main repository and expand it as implementation begins.

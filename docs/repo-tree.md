# Recommended repository tree

```text
.
├── AGENTS.md
├── CLAUDE.md
├── Makefile
├── README.md
├── .env.example
├── .gitignore
├── .claude
│   ├── agents
│   │   ├── README.md
│   │   ├── architecture-reviewer.md
│   │   ├── security-reviewer.md
│   │   └── eval-judge.md
│   ├── hooks
│   │   ├── README.md
│   │   ├── pre_tool_protect_files.sh
│   │   └── post_tool_go_checks.sh
│   ├── rules
│   └── skills
│       ├── admin-console-reports
│       │   └── SKILL.md
│       ├── agent-runtime-responses
│       │   └── SKILL.md
│       ├── api-contract-sse
│       │   └── SKILL.md
│       ├── docs-adr-runbook
│       │   └── SKILL.md
│       ├── eval-datasets-regression
│       │   └── SKILL.md
│       ├── mcp-adapter-factory
│       │   └── SKILL.md
│       ├── otel-langfuse-observability
│       │   └── SKILL.md
│       ├── postgres-sqlc-pgvector
│       │   └── SKILL.md
│       ├── repo-bootstrap-go
│       │   └── SKILL.md
│       ├── retrieval-ingest-provenance
│       │   └── SKILL.md
│       ├── security-tenancy-audit
│       │   └── SKILL.md
│       └── workflow-temporal-approval
│           └── SKILL.md
├── cmd
│   ├── api
│   │   └── main.go
│   └── worker
│       └── main.go
├── config
│   ├── default.yaml
│   └── local.yaml
├── db
│   ├── migrations
│   └── queries
├── docs
│   ├── adr
│   │   └── README.md
│   ├── runbooks
│   │   └── README.md
│   ├── architecture.md
│   ├── repo-tree.md
│   └── skills
│       └── README.md
├── eval
│   ├── datasets
│   ├── prompts
│   └── reports
├── internal
│   ├── agent
│   │   ├── AGENTS.override.md
│   │   ├── critic
│   │   ├── planner
│   │   ├── retrieval
│   │   └── tool
│   ├── app
│   ├── auth
│   ├── contextengine
│   ├── eval
│   │   └── AGENTS.override.md
│   ├── model
│   ├── observability
│   ├── retrieval
│   │   └── AGENTS.override.md
│   ├── session
│   ├── storage
│   ├── tools
│   │   ├── http
│   │   ├── mcp
│   │   └── registry
│   └── workflow
│       └── AGENTS.override.md
├── pkg
│   └── apierror
├── scripts
│   ├── dev
│   ├── hooks
│   └── sql
├── sqlc.yaml
└── web
    ├── admin
    │   └── AGENTS.override.md
    └── shared
```

Notes:
- This package does not generate the whole application codebase for you.
- It gives you the final instruction layer, skill playbooks, and the recommended tree to implement against.
- Create the missing code files incrementally with the relevant skill rather than scaffolding everything at once.

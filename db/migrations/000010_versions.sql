CREATE TABLE IF NOT EXISTS versions (
    id TEXT PRIMARY KEY,
    runtime_version TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    prompt_bundle TEXT NOT NULL DEFAULT '',
    planner_version TEXT NOT NULL DEFAULT '',
    retrieval_version TEXT NOT NULL DEFAULT '',
    tool_registry_version TEXT NOT NULL DEFAULT '',
    critic_version TEXT NOT NULL DEFAULT '',
    workflow_version TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

INSERT INTO versions (
    id,
    runtime_version,
    provider,
    model,
    prompt_bundle,
    planner_version,
    retrieval_version,
    tool_registry_version,
    critic_version,
    workflow_version,
    notes,
    created_at
) VALUES (
    'version-skeleton-2026-03-24',
    'runtime-skeleton-v1',
    '',
    '',
    'prompt-skeleton-v1',
    'planner-skeleton-v1',
    'retrieval-skeleton-v1',
    'ticket-http-adapters-v1',
    'critic-skeleton-v1',
    'temporal-bridge-v1',
    'Default runtime version for the current local skeleton.',
    TIMESTAMPTZ '2026-03-24 00:00:00+00'
)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS workflow_tasks (
    id                TEXT PRIMARY KEY,
    request_id        TEXT NOT NULL,
    tenant_id         TEXT NOT NULL,
    session_id        TEXT NOT NULL,
    task_type         TEXT NOT NULL,
    status            TEXT NOT NULL,
    reason            TEXT NOT NULL,
    error_reason      TEXT NOT NULL DEFAULT '',
    audit_ref         TEXT NOT NULL DEFAULT '',
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_tasks_tenant_status_created_at
    ON workflow_tasks (tenant_id, status, created_at DESC);

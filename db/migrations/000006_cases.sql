CREATE TABLE IF NOT EXISTS cases (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    status TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    source_task_id TEXT REFERENCES workflow_tasks(id) ON DELETE SET NULL,
    source_report_id TEXT REFERENCES reports(id) ON DELETE SET NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cases_tenant_updated_at
    ON cases (tenant_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_cases_source_task_id
    ON cases (source_task_id);

CREATE INDEX IF NOT EXISTS idx_cases_source_report_id
    ON cases (source_report_id);

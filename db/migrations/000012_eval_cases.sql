CREATE TABLE IF NOT EXISTS eval_cases (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    source_case_id TEXT NOT NULL UNIQUE REFERENCES cases(id) ON DELETE CASCADE,
    source_task_id TEXT REFERENCES workflow_tasks(id) ON DELETE SET NULL,
    source_report_id TEXT REFERENCES reports(id) ON DELETE SET NULL,
    trace_id TEXT NOT NULL DEFAULT '',
    version_id TEXT REFERENCES versions(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    operator_note TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_eval_cases_tenant_created_at
    ON eval_cases (tenant_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_eval_cases_version_id
    ON eval_cases (version_id);

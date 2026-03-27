CREATE TABLE IF NOT EXISTS eval_reports (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    run_id TEXT NOT NULL UNIQUE,
    dataset_id TEXT NOT NULL,
    dataset_name TEXT NOT NULL,
    run_status TEXT NOT NULL,
    status TEXT NOT NULL,
    summary TEXT NOT NULL,
    total_items INTEGER NOT NULL DEFAULT 0,
    recorded_results INTEGER NOT NULL DEFAULT 0,
    passed_items INTEGER NOT NULL DEFAULT 0,
    failed_items INTEGER NOT NULL DEFAULT 0,
    missing_results INTEGER NOT NULL DEFAULT 0,
    average_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    judge_version TEXT NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    bad_cases_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    ready_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_eval_reports_tenant_ready
    ON eval_reports (tenant_id, ready_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_eval_reports_run_id
    ON eval_reports (run_id);

CREATE TABLE IF NOT EXISTS eval_runs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL REFERENCES eval_datasets(id) ON DELETE RESTRICT,
    dataset_name TEXT NOT NULL,
    dataset_item_count INT NOT NULL,
    status TEXT NOT NULL,
    created_by TEXT NOT NULL,
    error_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_eval_runs_tenant_status_updated_at
    ON eval_runs (tenant_id, status, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_eval_runs_dataset_id
    ON eval_runs (dataset_id);

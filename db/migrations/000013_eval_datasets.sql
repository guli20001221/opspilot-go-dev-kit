CREATE TABLE IF NOT EXISTS eval_datasets (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_eval_datasets_tenant_status_created_at
    ON eval_datasets (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS eval_dataset_items (
    dataset_id TEXT NOT NULL REFERENCES eval_datasets(id) ON DELETE CASCADE,
    eval_case_id TEXT NOT NULL REFERENCES eval_cases(id) ON DELETE CASCADE,
    position INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dataset_id, eval_case_id),
    UNIQUE (dataset_id, position)
);

CREATE INDEX IF NOT EXISTS idx_eval_dataset_items_eval_case
    ON eval_dataset_items (eval_case_id);

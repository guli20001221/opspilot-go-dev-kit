ALTER TABLE eval_datasets
    ADD COLUMN IF NOT EXISTS published_by TEXT,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_eval_datasets_tenant_status_updated_at
    ON eval_datasets (tenant_id, status, updated_at DESC, id DESC);

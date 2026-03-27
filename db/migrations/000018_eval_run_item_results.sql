CREATE TABLE IF NOT EXISTS eval_run_item_results (
    run_id TEXT NOT NULL REFERENCES eval_runs(id) ON DELETE CASCADE,
    eval_case_id TEXT NOT NULL REFERENCES eval_cases(id),
    status TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (run_id, eval_case_id)
);

CREATE INDEX IF NOT EXISTS idx_eval_run_item_results_run_updated
    ON eval_run_item_results (run_id, updated_at, eval_case_id);

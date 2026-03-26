CREATE TABLE IF NOT EXISTS eval_run_items (
    run_id TEXT NOT NULL REFERENCES eval_runs(id) ON DELETE CASCADE,
    eval_case_id TEXT NOT NULL REFERENCES eval_cases(id),
    position INT NOT NULL,
    title TEXT NOT NULL,
    source_case_id TEXT NOT NULL REFERENCES cases(id),
    source_task_id TEXT,
    source_report_id TEXT,
    trace_id TEXT NOT NULL,
    version_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (run_id, position),
    UNIQUE (run_id, eval_case_id)
);

CREATE INDEX IF NOT EXISTS idx_eval_run_items_run_position ON eval_run_items (run_id, position);

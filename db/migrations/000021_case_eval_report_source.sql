ALTER TABLE cases
ADD COLUMN IF NOT EXISTS source_eval_report_id TEXT REFERENCES eval_reports(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_cases_source_eval_report_id
    ON cases (source_eval_report_id);

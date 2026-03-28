ALTER TABLE cases
    ADD COLUMN IF NOT EXISTS compare_left_eval_report_id TEXT REFERENCES eval_reports(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS compare_right_eval_report_id TEXT REFERENCES eval_reports(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS compare_selected_side TEXT NOT NULL DEFAULT '';

ALTER TABLE cases
    DROP CONSTRAINT IF EXISTS chk_cases_compare_selected_side;

ALTER TABLE cases
    ADD CONSTRAINT chk_cases_compare_selected_side
    CHECK (compare_selected_side IN ('', 'left', 'right'));

CREATE INDEX IF NOT EXISTS idx_cases_compare_left_eval_report_id
    ON cases (compare_left_eval_report_id);

CREATE INDEX IF NOT EXISTS idx_cases_compare_right_eval_report_id
    ON cases (compare_right_eval_report_id);

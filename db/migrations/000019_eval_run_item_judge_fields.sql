ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS verdict TEXT NOT NULL DEFAULT '';

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS score DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS judge_version TEXT NOT NULL DEFAULT '';

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS judge_output JSONB NOT NULL DEFAULT '{}'::jsonb;

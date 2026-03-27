ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS verdict TEXT NOT NULL DEFAULT '';

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS score DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS judge_version TEXT NOT NULL DEFAULT '';

ALTER TABLE eval_run_item_results
    ADD COLUMN IF NOT EXISTS judge_output JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE eval_run_item_results
SET
    verdict = CASE
        WHEN status = 'succeeded' THEN 'pass'
        ELSE 'fail'
    END,
    score = CASE
        WHEN status = 'succeeded' THEN 1
        ELSE 0
    END,
    judge_version = 'placeholder-v1',
    judge_output = jsonb_build_object(
        'judge_kind', 'placeholder',
        'judge_version', 'placeholder-v1',
        'verdict', CASE
            WHEN status = 'succeeded' THEN 'pass'
            ELSE 'fail'
        END,
        'score', CASE
            WHEN status = 'succeeded' THEN 1
            ELSE 0
        END,
        'rationale', detail
    )
WHERE verdict = ''
   OR judge_version = ''
   OR judge_output = '{}'::jsonb
   OR (status = 'succeeded' AND score = 0);

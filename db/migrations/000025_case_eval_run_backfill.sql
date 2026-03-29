UPDATE cases AS c
SET source_eval_run_id = matched.run_id
FROM (
    SELECT
        candidate.id AS case_id,
        runs.id AS run_id
    FROM cases AS candidate
    JOIN eval_runs AS runs
      ON runs.id = substring(candidate.summary FROM 'Follow up eval run ([^ ]+) result for')
     AND runs.tenant_id = candidate.tenant_id
    JOIN eval_run_items AS items
      ON items.run_id = runs.id
     AND items.eval_case_id = candidate.source_eval_case_id
    WHERE candidate.source_eval_run_id IS NULL
      AND candidate.source_eval_case_id IS NOT NULL
      AND candidate.summary LIKE 'Follow up eval run % result for %'
) AS matched
WHERE c.id = matched.case_id
  AND c.source_eval_run_id IS NULL;

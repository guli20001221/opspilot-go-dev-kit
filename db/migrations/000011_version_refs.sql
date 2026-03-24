ALTER TABLE workflow_tasks
    ADD COLUMN IF NOT EXISTS version_id TEXT REFERENCES versions(id);

ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS version_id TEXT REFERENCES versions(id);

UPDATE workflow_tasks
SET version_id = 'version-skeleton-2026-03-24'
WHERE version_id IS NULL;

UPDATE reports
SET version_id = 'version-skeleton-2026-03-24'
WHERE version_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_workflow_tasks_version_id
    ON workflow_tasks (version_id);

CREATE INDEX IF NOT EXISTS idx_reports_version_id
    ON reports (version_id);

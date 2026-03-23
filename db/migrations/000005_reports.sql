CREATE TABLE IF NOT EXISTS reports (
  id                  TEXT PRIMARY KEY,
  tenant_id           TEXT NOT NULL,
  source_task_id      TEXT NOT NULL REFERENCES workflow_tasks(id) ON DELETE CASCADE,
  report_type         TEXT NOT NULL,
  status              TEXT NOT NULL,
  title               TEXT NOT NULL,
  summary             TEXT NOT NULL DEFAULT '',
  content_uri         TEXT NOT NULL DEFAULT '',
  metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by          TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL,
  ready_at            TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS reports_source_task_id_idx
  ON reports (source_task_id);

CREATE INDEX IF NOT EXISTS reports_tenant_created_idx
  ON reports (tenant_id, created_at DESC);

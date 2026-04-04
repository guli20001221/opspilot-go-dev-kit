CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS retrieval_chunks (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT NOT NULL,
    document_id       TEXT NOT NULL,
    document_version  INT  NOT NULL DEFAULT 1,
    chunk_id          TEXT NOT NULL,
    source_title      TEXT NOT NULL DEFAULT '',
    source_uri        TEXT NOT NULL DEFAULT '',
    snippet           TEXT NOT NULL DEFAULT '',
    embedding         vector(1536) NOT NULL,
    permissions_scope TEXT NOT NULL DEFAULT '',
    published_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, chunk_id)
);

CREATE INDEX IF NOT EXISTS idx_retrieval_chunks_tenant
    ON retrieval_chunks (tenant_id);

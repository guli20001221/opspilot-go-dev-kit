ALTER TABLE retrieval_chunks
    ADD COLUMN IF NOT EXISTS parent_chunk_id TEXT,
    ADD COLUMN IF NOT EXISTS context_prefix TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS search_tsv TSVECTOR;

CREATE INDEX IF NOT EXISTS idx_retrieval_chunks_search_tsv
    ON retrieval_chunks USING GIN (search_tsv);

CREATE INDEX IF NOT EXISTS idx_retrieval_chunks_parent
    ON retrieval_chunks (document_id, parent_chunk_id)
    WHERE parent_chunk_id IS NOT NULL;

CREATE OR REPLACE FUNCTION retrieval_chunks_tsv_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search_tsv := to_tsvector('english', COALESCE(NEW.context_prefix, '') || ' ' || COALESCE(NEW.snippet, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_retrieval_chunks_tsv ON retrieval_chunks;
CREATE TRIGGER trg_retrieval_chunks_tsv
    BEFORE INSERT OR UPDATE OF context_prefix, snippet
    ON retrieval_chunks
    FOR EACH ROW
    EXECUTE FUNCTION retrieval_chunks_tsv_trigger();

UPDATE retrieval_chunks
SET search_tsv = to_tsvector('english', COALESCE(context_prefix, '') || ' ' || COALESCE(snippet, ''))
WHERE search_tsv IS NULL;

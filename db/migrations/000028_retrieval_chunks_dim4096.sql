-- Resize embedding column from vector(1536) to vector(4096) to match
-- doubao-embedding-large output dimension.
ALTER TABLE retrieval_chunks
    ALTER COLUMN embedding TYPE vector(4096);

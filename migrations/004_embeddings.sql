CREATE TABLE embedding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    search_chunk_id UUID NOT NULL REFERENCES search_chunk(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    dimensions INTEGER NOT NULL CHECK (dimensions = 1536),
    content_sha256 TEXT NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (search_chunk_id, provider, model)
);

CREATE INDEX embedding_search_chunk_id_idx ON embedding(search_chunk_id);
CREATE INDEX embedding_cosine_idx ON embedding USING hnsw (embedding vector_cosine_ops);

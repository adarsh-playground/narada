CREATE TABLE ask_interaction (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scripture_id UUID REFERENCES scripture(id) ON DELETE SET NULL,
    question TEXT NOT NULL CHECK (btrim(question) <> ''),
    answer_text TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    error_message TEXT,
    embedding_provider TEXT NOT NULL DEFAULT 'openai',
    embedding_model TEXT NOT NULL,
    answer_provider TEXT NOT NULL DEFAULT 'openai',
    answer_model TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    embedding_input_tokens INTEGER NOT NULL DEFAULT 0 CHECK (embedding_input_tokens >= 0),
    answer_input_tokens INTEGER NOT NULL DEFAULT 0 CHECK (answer_input_tokens >= 0),
    answer_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK (answer_output_tokens >= 0),
    embedding_price_nanos_per_token BIGINT NOT NULL CHECK (embedding_price_nanos_per_token >= 0),
    answer_input_price_nanos_per_token BIGINT NOT NULL CHECK (answer_input_price_nanos_per_token >= 0),
    answer_output_price_nanos_per_token BIGINT NOT NULL CHECK (answer_output_price_nanos_per_token >= 0),
    embedding_cost_nanos BIGINT NOT NULL DEFAULT 0 CHECK (embedding_cost_nanos >= 0),
    answer_input_cost_nanos BIGINT NOT NULL DEFAULT 0 CHECK (answer_input_cost_nanos >= 0),
    answer_output_cost_nanos BIGINT NOT NULL DEFAULT 0 CHECK (answer_output_cost_nanos >= 0),
    total_cost_usd NUMERIC(16, 10) GENERATED ALWAYS AS
        ((embedding_cost_nanos + answer_input_cost_nanos + answer_output_cost_nanos)::numeric / 1000000000) STORED,
    duration_ms INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    feedback TEXT CHECK (feedback IS NULL OR feedback IN ('helpful', 'not_helpful')),
    feedback_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX ask_interaction_created_at_idx ON ask_interaction(created_at DESC);
CREATE INDEX ask_interaction_status_idx ON ask_interaction(status);

CREATE TABLE ask_interaction_evidence (
    ask_interaction_id UUID NOT NULL REFERENCES ask_interaction(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL CHECK (rank > 0),
    search_chunk_id UUID REFERENCES search_chunk(id) ON DELETE SET NULL,
    kind TEXT NOT NULL,
    citation_label TEXT NOT NULL,
    source_name TEXT,
    text_snapshot TEXT NOT NULL,
    verse_references JSONB NOT NULL DEFAULT '[]'::jsonb,
    similarity DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (ask_interaction_id, rank)
);

CREATE INDEX ask_interaction_evidence_chunk_idx ON ask_interaction_evidence(search_chunk_id);

CREATE TABLE search_chunk (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scripture_id UUID NOT NULL REFERENCES scripture(id) ON DELETE CASCADE,
    source_id UUID REFERENCES source(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('verse_translation', 'commentary')),
    stable_key TEXT NOT NULL,
    citation_label TEXT NOT NULL,
    text TEXT NOT NULL CHECK (btrim(text) <> ''),
    content_sha256 TEXT NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    builder_version TEXT NOT NULL,
    build_token UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scripture_id, stable_key)
);

CREATE INDEX search_chunk_scripture_kind_idx ON search_chunk(scripture_id, kind);
CREATE INDEX search_chunk_source_id_idx ON search_chunk(source_id);

CREATE TABLE search_chunk_verse (
    search_chunk_id UUID NOT NULL REFERENCES search_chunk(id) ON DELETE CASCADE,
    verse_id UUID NOT NULL REFERENCES verse(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position > 0),
    PRIMARY KEY (search_chunk_id, verse_id),
    UNIQUE (search_chunk_id, position)
);

CREATE INDEX search_chunk_verse_verse_id_idx ON search_chunk_verse(verse_id);

CREATE TABLE search_chunk_commentary (
    search_chunk_id UUID PRIMARY KEY REFERENCES search_chunk(id) ON DELETE CASCADE,
    commentary_id UUID NOT NULL UNIQUE REFERENCES commentary(id) ON DELETE CASCADE
);

CREATE TABLE search_chunk_build (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scripture_id UUID NOT NULL REFERENCES scripture(id) ON DELETE CASCADE,
    builder_version TEXT NOT NULL,
    chunk_count INTEGER NOT NULL CHECK (chunk_count > 0),
    built_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

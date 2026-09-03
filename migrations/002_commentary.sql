CREATE TABLE commentary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES source(id) ON DELETE RESTRICT,
    chapter_id UUID NOT NULL REFERENCES chapter(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (btrim(text) <> ''),
    sequence_number INTEGER NOT NULL CHECK (sequence_number > 0),
    citation_label TEXT,
    UNIQUE (source_id, chapter_id, sequence_number)
);

CREATE INDEX commentary_source_id_idx ON commentary(source_id);
CREATE INDEX commentary_chapter_id_idx ON commentary(chapter_id);

CREATE TABLE commentary_verse (
    commentary_id UUID NOT NULL REFERENCES commentary(id) ON DELETE CASCADE,
    verse_id UUID NOT NULL REFERENCES verse(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL DEFAULT 'primary',
    position INTEGER NOT NULL CHECK (position > 0),
    PRIMARY KEY (commentary_id, verse_id)
);

CREATE INDEX commentary_verse_verse_id_idx ON commentary_verse(verse_id);

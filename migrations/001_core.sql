CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE scripture (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    short_name TEXT NOT NULL UNIQUE,
    original_language TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chapter (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scripture_id UUID NOT NULL REFERENCES scripture(id) ON DELETE CASCADE,
    number INTEGER NOT NULL CHECK (number > 0),
    source_external_id TEXT,
    title TEXT,
    original_title TEXT,
    transliterated_title TEXT,
    meaning TEXT,
    summary TEXT,
    summary_hindi TEXT,
    expected_verse_count INTEGER CHECK (expected_verse_count > 0),
    UNIQUE (scripture_id, number)
);

CREATE INDEX chapter_scripture_id_idx ON chapter(scripture_id);

CREATE TABLE verse (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID NOT NULL REFERENCES chapter(id) ON DELETE CASCADE,
    verse_number TEXT NOT NULL,
    sequence_number INTEGER NOT NULL CHECK (sequence_number > 0),
    scripture_sequence_number INTEGER NOT NULL CHECK (scripture_sequence_number > 0),
    source_external_id TEXT,
    original_text TEXT NOT NULL CHECK (btrim(original_text) <> ''),
    transliteration TEXT,
    word_meanings TEXT,
    UNIQUE (chapter_id, verse_number),
    UNIQUE (chapter_id, sequence_number),
    UNIQUE (chapter_id, scripture_sequence_number)
);

CREATE INDEX verse_chapter_id_idx ON verse(chapter_id);
CREATE INDEX verse_scripture_sequence_number_idx ON verse(scripture_sequence_number);

CREATE TABLE source (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('translation', 'commentary')),
    tradition TEXT,
    language TEXT NOT NULL,
    publication TEXT,
    description TEXT,
    source_url TEXT,
    license TEXT,
    UNIQUE (name, type)
);

CREATE TABLE translation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    verse_id UUID NOT NULL REFERENCES verse(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES source(id) ON DELETE RESTRICT,
    text TEXT NOT NULL CHECK (btrim(text) <> ''),
    UNIQUE (verse_id, source_id)
);

CREATE INDEX translation_verse_id_idx ON translation(verse_id);
CREATE INDEX translation_source_id_idx ON translation(source_id);

CREATE TABLE data_import (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset TEXT NOT NULL,
    source_url TEXT NOT NULL,
    source_version TEXT,
    license TEXT,
    importer_version TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    chapter_count INTEGER NOT NULL,
    verse_count INTEGER NOT NULL
);

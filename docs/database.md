# Database Design

## Overview

Narada is built around the principle of **separating knowledge from reasoning**.

The database stores authoritative knowledge from scriptures, translations, commentaries, and concepts. AI models use this information to reason and generate responses but are never considered the source of truth.

The schema is designed to support:

* Multiple scriptures
* Multiple source languages
* Multiple translations
* Multiple commentaries
* Semantic search
* Concept-based retrieval
* Future expansion beyond the Bhagavad Gita

---

# Entity Relationship

```text
Scripture
    └── Chapter
            └── Verse
                    ├── Translation(s)
                    ├── Commentary(s)
                    ├── Embedding(s)
                    └── Concept(s)
```

---

# scripture

Represents a canonical work.

Examples

* Bhagavad Gita
* Isha Upanishad
* Katha Upanishad
* Yoga Sutras
* Dhammapada

| Column            | Type      | Description                 |
| ----------------- | --------- | --------------------------- |
| id                | UUID      | Primary key                 |
| name              | Text      | Scripture name              |
| short_name        | Text      | Abbreviated name            |
| original_language | Text      | Sanskrit, Pali, Tamil, etc. |
| description       | Text      | Optional summary            |
| created_at        | Timestamp | Record creation time        |

---

# chapter

Represents a chapter within a scripture.

| Column       | Type    | Description      |
| ------------ | ------- | ---------------- |
| id           | UUID    | Primary key      |
| scripture_id | UUID    | FK to scripture  |
| number       | Integer | Chapter number   |
| title        | Text    | Optional title   |
| summary      | Text    | Optional summary |

---

# verse

Stores the original scripture text.

| Column          | Type    | Description                |
| --------------- | ------- | -------------------------- |
| id              | UUID    | Primary key                |
| chapter_id      | UUID    | FK to chapter              |
| verse_number    | Integer | Verse number               |
| original_text   | Text    | Original scripture text    |
| transliteration | Text    | Transliteration (optional) |

The original text is immutable and represents the authoritative source.

---

# source

Represents the origin of a translation or commentary.

Examples

Translations

* Swami Gambhirananda
* Eknath Easwaran
* Swami Chinmayananda

Commentaries

* Shankaracharya
* Ramanuja
* Madhva
* Prabhupada

| Column      | Type | Description                      |
| ----------- | ---- | -------------------------------- |
| id          | UUID | Primary key                      |
| name        | Text | Person or publication            |
| type        | Enum | translation, commentary          |
| tradition   | Text | Optional philosophical tradition |
| language    | Text | Language of the source           |
| publication | Text | Optional publication details     |
| description | Text | Optional biography or notes      |

---

# translation

Represents one translation of a verse.

A verse may have many translations.

| Column    | Type | Description  |
| --------- | ---- | ------------ |
| id        | UUID | Primary key  |
| verse_id  | UUID | FK to verse  |
| source_id | UUID | FK to source |
| text      | Text | Translation  |

---

# commentary

Represents an interpretation of a verse.

A verse may have many commentaries.

| Column    | Type | Description  |
| --------- | ---- | ------------ |
| id        | UUID | Primary key  |
| verse_id  | UUID | FK to verse  |
| source_id | UUID | FK to source |
| text      | Text | Commentary   |

Commentaries explain the philosophical meaning of a verse rather than translating it.

---

# concept

Represents philosophical concepts.

Examples

* Duty
* Karma
* Devotion
* Fear
* Attachment
* Knowledge

| Column      | Type | Description                 |
| ----------- | ---- | --------------------------- |
| id          | UUID | Primary key                 |
| name        | Text | Concept name                |
| description | Text | Optional explanation        |
| parent_id   | UUID | Self-reference for taxonomy |

Example hierarchy

```text
Fear
    ├── Fear of Failure
    ├── Fear of Death
    └── Fear of Loss
```

---

# verse_concept

Maps verses to concepts.

Many-to-many relationship.

| Column     | Type    | Description              |
| ---------- | ------- | ------------------------ |
| verse_id   | UUID    | FK to verse              |
| concept_id | UUID    | FK to concept            |
| confidence | Decimal | Optional relevance score |

A verse may belong to many concepts.

A concept may reference many verses.

---

# embedding

Stores vector embeddings used for semantic search.

Embeddings may be generated for

* Verse
* Translation
* Commentary
* Search chunk

| Column      | Type      | Description                           |
| ----------- | --------- | ------------------------------------- |
| id          | UUID      | Primary key                           |
| entity_type | Text      | verse, translation, commentary, chunk |
| entity_id   | UUID      | Referenced entity                     |
| model       | Text      | Embedding model                       |
| dimensions  | Integer   | Vector dimensions                     |
| embedding   | Vector    | pgvector column                       |
| created_at  | Timestamp | Creation timestamp                    |

Embeddings are implementation data and can be regenerated at any time.

---

# Search Chunk (Logical Entity)

The application does not necessarily search verses directly.

Instead, a searchable chunk may consist of

* Original verse
* Translation
* Commentary
* Metadata

The chunk is converted into an embedding and stored in the embedding table.

Example

```text
BG 2.47

Original Text

Translation

Commentary

Metadata:
Duty
Action
Attachment
Karma Yoga
```

---

# Future Tables

## question_log

Stores anonymous user questions.

Purpose

* Improve retrieval
* Discover missing concepts
* Build evaluation datasets

---

## evaluation

Stores benchmark questions.

Example

Question

"I am afraid of failure."

Expected Verses

* BG 2.47
* BG 2.48

Used to measure retrieval quality over time.

---

## study_collection

Curated reading plans.

Examples

* Karma Yoga
* Bhakti Yoga
* Leadership
* Fear and Anxiety

---

## bookmark

Allows users to bookmark verses.

---

## note

Allows users to create personal notes on verses and commentaries.

---

# MVP Schema

The first version of Narada requires only the following tables:

* scripture
* chapter
* verse
* source
* translation
* commentary
* embedding

The concept taxonomy, user features, and evaluation framework can be added incrementally without requiring changes to the core schema.

---

# Design Principles

1. Original scripture text is immutable.
2. Translations and commentaries are stored independently.
3. A verse can have unlimited translations and commentaries.
4. Knowledge is independent of AI models.
5. Embeddings are derived data and may be regenerated.
6. The schema is language-agnostic and supports multiple scriptures.
7. The schema is designed for long-term extensibility while keeping the MVP simple.

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
                    ├── Commentary passage(s), via commentary_verse
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

| Column          | Type    | Description                                                |
| --------------- | ------- | ---------------------------------------------------------- |
| id              | UUID    | Primary key                                                |
| chapter_id      | UUID    | FK to chapter                                              |
| verse_number    | Text    | Canonical label, such as `1`, `13.1a`, or an edition label |
| sequence_number | Integer | Sort order of the verse within the chapter                 |
| original_text   | Text    | Original scripture text                                    |
| transliteration | Text    | Transliteration (optional)                                 |
| word_meanings   | Text    | Source-provided word-by-word meanings (optional)           |

The original text is immutable and represents the authoritative source.

`verse_number` preserves the canonical identifier without assuming that every
scripture or edition uses whole-number labels. `sequence_number` provides stable
numeric ordering independently of the displayed label. The combination of
`chapter_id` and `verse_number` must be unique, as must the combination of
`chapter_id` and `sequence_number`.

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

Represents a distinct passage of interpretation from a source.

A commentary passage is stored once, even when it discusses multiple verses.
The relationship between commentary passages and verses is defined by
`commentary_verse`.

| Column          | Type    | Description                                      |
| --------------- | ------- | ------------------------------------------------ |
| id              | UUID    | Primary key                                      |
| source_id       | UUID    | FK to source                                     |
| text            | Text    | Commentary passage                               |
| sequence_number | Integer | Ordering within the source, chapter, or section  |
| citation_label  | Text    | Optional human-readable source or section label  |

Commentaries explain the philosophical meaning of a verse rather than translating it.

---

# commentary_verse

Maps commentary passages to the verses they discuss.

This is a many-to-many relationship:

* A commentary passage may discuss one verse, several verses, or a verse range.
* A verse may be discussed by many commentary passages and sources.
* Commentary text is not duplicated when it applies to multiple verses.

| Column        | Type    | Description                                             |
| ------------- | ------- | ------------------------------------------------------- |
| commentary_id | UUID    | FK to commentary                                        |
| verse_id      | UUID    | FK to verse                                             |
| relation_type | Text    | primary, referenced, context, or another defined value  |
| position      | Integer | Order of the verse within the commentary passage        |

The composite key `(commentary_id, verse_id)` prevents duplicate links.

Example:

```text
Commentary passage: explanation of Bhagavad Gita 2.47–2.49

commentary_verse
    ├── BG 2.47 (primary, position 1)
    ├── BG 2.48 (primary, position 2)
    └── BG 2.49 (primary, position 3)
```

---

# search_chunk

Stores rebuildable units used by semantic retrieval. A chunk is not an
authoritative source: its text is derived from a translation or commentary and
retains links back to those records.

| Column          | Description                                      |
| --------------- | ------------------------------------------------ |
| kind            | `verse_translation` or `commentary`              |
| stable_key      | Deterministic identity used for idempotent builds |
| citation_label  | Human-readable reference such as `BG 2.47`        |
| text            | Text that will later be embedded                  |
| content_sha256  | Detects content changes and stale embeddings      |
| builder_version | Identifies the chunking strategy                  |

`search_chunk_verse` links every chunk to its cited verses.
`search_chunk_commentary` links a commentary chunk to its authoritative
commentary passage. Commentary applying to several verses is stored as one
chunk with multiple verse links.

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

| Column          | Type         | Description                              |
| --------------- | ------------ | ---------------------------------------- |
| id              | UUID         | Primary key                              |
| search_chunk_id | UUID         | Chunk represented by the vector          |
| provider        | Text         | Embedding provider                       |
| model           | Text         | Embedding model                          |
| dimensions      | Integer      | Vector dimensions                        |
| content_sha256  | Text         | Chunk hash used to detect stale vectors  |
| embedding       | Vector(1536) | pgvector value                           |
| created_at      | Timestamp    | Creation timestamp                       |
| updated_at      | Timestamp    | Last replacement time                    |

Embeddings are implementation data and can be regenerated at any time.

---

# data_import

Records the provenance of each completed dataset import.

| Column           | Type      | Description                              |
| ---------------- | --------- | ---------------------------------------- |
| id               | UUID      | Primary key                              |
| dataset          | Text      | Stable dataset name                      |
| source_url       | Text      | Location of the upstream source          |
| source_version   | Text      | Commit hash, release, or snapshot label  |
| license          | Text      | License identifier or note               |
| importer_version | Text      | Version of the importer                  |
| imported_at      | Timestamp | Completion time                          |
| chapter_count    | Integer   | Number of chapters processed             |
| verse_count      | Integer   | Number of verses processed               |

An import record describes the dataset operation, not a user action. Repeated
idempotent imports create separate audit records while leaving canonical chapter
and verse rows unchanged.

---

# Anonymous Question and Answer History

## question

Stores each question submitted to Narada.

Questions are anonymous. This table must not contain a user ID, account ID,
session ID, IP address, device identifier, or other field intended to identify
the person asking the question.

| Column     | Type      | Description                                  |
| ---------- | --------- | -------------------------------------------- |
| id         | UUID      | Primary key                                  |
| text       | Text      | Question as submitted                        |
| language   | Text      | Detected or supplied language, when known    |
| created_at | Timestamp | Submission time                              |

Anonymous question history can be used to improve retrieval, discover missing
concepts, and build evaluation datasets. Operational logs must follow the same
privacy rule and must not add identity or network identifiers to these records.

---

## answer

Stores an answer-generation attempt for a question. A question may have more
than one answer if it is retried or regenerated.

| Column         | Type      | Description                                      |
| -------------- | --------- | ------------------------------------------------ |
| id             | UUID      | Primary key                                      |
| question_id    | UUID      | FK to question                                   |
| text           | Text      | Generated answer; nullable for failed attempts   |
| status         | Text      | pending, completed, insufficient_evidence, failed |
| model          | Text      | Answer-generation model                          |
| prompt_version | Text      | Version of the prompt or answer strategy         |
| latency_ms     | Integer   | End-to-end generation time                       |
| error_code     | Text      | Safe internal category for a failed attempt      |
| created_at     | Timestamp | Attempt creation time                            |
| completed_at   | Timestamp | Completion time                                  |

`error_code` stores a category rather than a raw provider error that might
accidentally contain question text or request metadata.

---

## answer_evidence

Records the retrieved search chunks considered for an answer. It provides an
audit trail for retrieval, grounding, and citations.

| Column          | Type    | Description                                      |
| --------------- | ------- | ------------------------------------------------ |
| answer_id       | UUID    | FK to answer                                     |
| search_chunk_id | UUID    | FK to the retrieved search chunk                 |
| retrieval_rank  | Integer | Rank before answer generation                    |
| retrieval_score | Decimal | Similarity or reranking score                    |
| included        | Boolean | Whether the chunk was supplied to the model      |
| cited           | Boolean | Whether the completed answer cited this evidence |
| citation_order  | Integer | Display order when cited                         |

The composite key `(answer_id, search_chunk_id)` prevents duplicate evidence
records. A cited record must also have `included = true`; the application must
not cite evidence that was not supplied during answer generation.

Together these tables preserve:

* What was asked
* What the system answered
* Which model and prompt strategy were used
* Which evidence was retrieved, selected, and cited
* Whether the attempt completed or failed

They intentionally do not preserve who asked the question.

---

# Future Tables

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
* commentary_verse
* search_chunk
* embedding
* data_import
* question
* answer
* answer_evidence

The concept taxonomy, user features, and evaluation framework can be added incrementally without requiring changes to the core schema.

---

# Design Principles

1. Original scripture text is immutable.
2. Translations and commentaries are stored independently.
3. A verse can have unlimited translations and commentaries.
4. Knowledge is independent of AI models.
5. Embeddings are derived data and may be regenerated.
6. Questions and answers are stored anonymously without user identity fields.
6. The schema is language-agnostic and supports multiple scriptures.
7. The schema is designed for long-term extensibility while keeping the MVP simple.
## Ask history and cost audit

`ask_interaction` records every valid Ask request, including completed and
failed requests. It stores the question, grounded answer, model and prompt
versions, exact token usage, the price snapshot used for calculation, request
duration, and costs. Costs are held as integer billionths of a US dollar to
avoid rounding sub-cent requests; `total_cost_usd` is generated for display.

`ask_interaction_evidence` preserves the ranked passages supplied to the answer
model. Citation, source, text, verse references, and similarity are snapshotted
so a historical answer remains auditable after the retrieval corpus changes.

Inspect recent requests in DBeaver:

```sql
SELECT created_at, status, question, embedding_input_tokens,
       answer_input_tokens, answer_output_tokens, total_cost_usd,
       duration_ms, answer_model
FROM ask_interaction
ORDER BY created_at DESC;
```

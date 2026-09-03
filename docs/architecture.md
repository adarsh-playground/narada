# Narada Architecture

## Purpose

Narada is a source-grounded question-answering system for the Bhagavad Gita.
It uses language models to understand questions and explain retrieved material,
while scripture, translations, and commentaries remain the authoritative sources.

The first release focuses on the Bhagavad Gita and one commentary. The
architecture should support additional translations, commentators, and
scriptures without requiring a redesign of the core system.

## Architectural Principles

1. **Knowledge and reasoning are separate.** Models reason over retrieved source
   material but do not serve as the source of scriptural claims.
2. **Answers are traceable.** Every substantive answer must cite the verses and
   commentary passages used to produce it.
3. **Source text is preserved.** Imported text is stored with its provenance and
   is not silently rewritten by a model.
4. **Derived data is replaceable.** Search chunks, embeddings, and model-generated
   concept mappings can be rebuilt from the stored source material.
5. **The MVP stays simple.** New infrastructure and abstractions are introduced
   only when required by the product.

## System Context

```text
User
  |
  v
React Web Application
  |
  v
Go API (Echo)
  |--------------------------|
  v                          v
PostgreSQL + pgvector     AI Providers
  ^                       - embeddings
  |                       - reranking (optional)
Ingestion Pipeline        - answer generation
  |
  v
Scripture and Commentary Files
```

## Components

### Web Application

The React application provides:

* Natural-language question entry
* Grounded answers with inline citations
* Verse and commentary source views
* Chapter and verse navigation
* Scholar selection when additional commentaries are added

The browser communicates only with the Go API. It does not call model providers
or access the database directly.

### Go API

The Go service, built with Echo, owns the application workflow and exposes HTTP
JSON endpoints. Its responsibilities include:

* Validating requests
* Loading chapters, verses, translations, and commentaries
* Running semantic and structured retrieval
* Constructing model prompts from retrieved evidence
* Calling AI providers
* Validating and returning citations
* Recording operational metadata needed for debugging and evaluation

The API is organized into domain, storage, retrieval, and AI-provider layers so
that model and embedding vendors can be changed without altering the knowledge
model.

### PostgreSQL and pgvector

PostgreSQL is the system of record for both authoritative and derived data.
pgvector provides semantic similarity search.

Authoritative data includes:

* Scriptures
* Chapters
* Verses
* Translations
* Commentary passages and their verse links
* Sources and provenance metadata

Derived data includes:

* Search chunks
* Embeddings
* Concept mappings generated or suggested by models

The logical data model is defined in [database.md](database.md).

### Ingestion Pipeline

Ingestion runs as an explicit command or job rather than inside normal API
requests. It must be repeatable and safe to rerun.

The initial pipeline has these stages:

```text
Raw source files
  -> parse
  -> normalize formatting
  -> validate chapter and verse references
  -> upsert authoritative records
  -> build search chunks
  -> generate embeddings
  -> record import provenance
```

The first load uses the existing chapter and verse JSON files. Commentary is
loaded later, after its extracted text is cleaned, segmented into passages, and
linked to one or more verses through `commentary_verse`.

Each import should record at least:

* Source name and URL
* License identifier or license note
* Source version or commit hash, when available
* Import timestamp
* Importer version

Generated embeddings must record the model name and dimensions so they can be
identified and rebuilt.

## Retrieval and Answer Flow

```text
1. Receive the user's question
2. Normalize the question and detect explicit verse references
3. Retrieve candidate chunks using:
   - direct chapter/verse lookup
   - semantic vector search
   - metadata filters such as scripture and commentator
4. Optionally rerank the candidates
5. Select a bounded evidence set
6. Ask the language model to answer only from that evidence
7. Validate that cited records were included in the evidence set
8. Return the answer, citations, and source excerpts
```

Direct reference lookup takes priority when a user names a chapter and verse.
Semantic retrieval handles indirect questions such as anxiety about failure or
attachment to outcomes.

The generated answer should distinguish among:

* The original verse
* A translation
* A commentator's interpretation
* The model's synthesis of the supplied evidence

If the retrieved evidence is insufficient, the system should say so rather than
present unsupported claims as scriptural teaching.

## Search Chunks

A search chunk is a derived retrieval unit, not an authoritative source. A chunk
may contain a verse, translation, commentary passage, or a carefully bounded
combination of them.

Every chunk must retain links to its contributing database records. Citations are
created from those links, never reconstructed from generated answer text.

For the MVP, prefer small independently citable chunks:

* One verse and its translation
* One commentary passage linked to one or more verses

This avoids retrieving an entire chapter when only a short passage is relevant.

## API Surface for the MVP

The initial API can remain small:

```text
GET  /api/v1/scriptures
GET  /api/v1/scriptures/:scripture/chapters
GET  /api/v1/scriptures/:scripture/chapters/:chapter/verses/:verse
POST /api/v1/questions
```

The question response should include a stable record identifier for each cited
verse or commentary passage so the client can open the underlying source.

## Configuration and Secrets

Runtime configuration is supplied through environment variables or a deployment
secret manager. Model credentials are available only to the Go service and
ingestion jobs. They are never sent to the browser or stored in source control.

Configuration includes:

* PostgreSQL connection string
* Answer-generation model
* Embedding model and dimensions
* Provider credentials
* Retrieval candidate and evidence limits

## Observability and Evaluation

Application logs should include request and retrieval identifiers but should not
record private user text by default. For each answer, the system should be able to
inspect:

* Retrieved chunk identifiers and scores
* Final evidence passed to the model
* Model and prompt version
* Citation-validation result
* Latency and provider errors

A small evaluation set of representative questions should be maintained before
tuning retrieval. It should test direct verse lookup, indirect concepts,
insufficient evidence, and citation correctness.

## Deployment Shape

The MVP requires three deployable concerns:

* A static React web application
* One Go API service
* One PostgreSQL database with pgvector

Ingestion can initially run as a Go CLI from the same codebase as the API. A
separate worker or queue is unnecessary until imports or embedding generation
need to run continuously.

## Deferred Decisions

The following are intentionally deferred until the MVP demonstrates a need:

* Separate author, work, publication, and edition tables
* A dedicated graph database for concepts
* Multiple API services or microservices
* A background job queue
* Model fine-tuning
* User accounts and personalized study history
* Streaming responses

## Initial Implementation Sequence

1. Create PostgreSQL migrations for scriptures, chapters, verses, and sources.
2. Implement a validated, idempotent loader for the existing Gita JSON files.
3. Add translations, commentary passages, and `commentary_verse` links.
4. Create search chunks and embeddings.
5. Implement retrieval with direct-reference and semantic-search paths.
6. Add grounded answer generation and citation validation.
7. Build the minimal React question-and-source interface.
8. Add an evaluation suite and tune retrieval against it.

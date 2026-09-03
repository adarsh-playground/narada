# Gita Data Loading

This guide explains how to load the Bhagavad Gita chapters and verses into the
local Narada PostgreSQL database.

The importer reads:

* `data/gita/chapters.json`
* `data/gita/verse.json`

Before writing anything, it validates that the dataset contains 18 chapters and
701 unique verses, that every chapter count matches its metadata, and that each
verse contains Sanskrit text and transliteration.

## Prerequisites

Run these commands from the repository root. You need:

* Go 1.26 or later
* Docker with Docker Compose

## First-Time Load

Start the local PostgreSQL and pgvector container:

```sh
docker compose up -d postgres
```

Set the database connection for the current terminal:

```sh
export DATABASE_URL='postgres://narada:narada@localhost:5432/narada?sslmode=disable'
```

Apply all database migrations:

```sh
go run ./cmd/migrate
```

Validate and import the Gita dataset:

```sh
go run ./cmd/import-gita
```

A successful run prints:

```text
validated 18 chapters and 701 verses
Gita import completed
```

## Running the Load Again

The migration command and importer are safe to rerun:

```sh
go run ./cmd/migrate
go run ./cmd/import-gita
```

Scripture, chapter, and verse records are upserted using their stable unique
keys. Existing canonical records are updated rather than duplicated. Each
successful run creates a separate `data_import` audit record.

## Recording the Source Version

When the upstream dataset version is known, record its commit hash or release:

```sh
go run ./cmd/import-gita -source-version '<commit-or-version>'
```

If this option is omitted, the importer records the source version as `unknown`.

## Validation Without a Database Write

Validate the JSON files without connecting to PostgreSQL:

```sh
go run ./cmd/import-gita -validate-only
```

Custom input paths can also be supplied:

```sh
go run ./cmd/import-gita \
  -chapters path/to/chapters.json \
  -verses path/to/verses.json \
  -validate-only
```

## Verify the Loaded Data

Check the top-level row counts:

```sh
docker compose exec postgres psql -U narada -d narada -c \
  "SELECT
     (SELECT count(*) FROM scripture) AS scriptures,
     (SELECT count(*) FROM chapter) AS chapters,
     (SELECT count(*) FROM verse) AS verses;"
```

The expected result is one scripture, 18 chapters, and 701 verses.

## Load The Holy Geeta Translation and Commentary

All 18 chapters are extracted from PDF typography: italic Book Antiqua lines are verse
translations, and the following regular-font prose is commentary. Commentary
following a group of verses is stored once and linked to every verse in that
group.

After loading the canonical Gita data, parse and import Chapter 1 with:

```sh
make load-chinmayananda
```

To inspect the extracted JSON without touching PostgreSQL:

```sh
python3 data/gita/commentaries/Chinmayananda/parse_chapter.py
go run ./cmd/import-chinmayananda -validate-only
```

The parser validates the result against all 701 canonical verses before writing
each chapter JSON file. Both the parser output and database import are
deterministic and safe to rerun.

## Build Search Chunks

After importing the Gita and commentary, build the derived retrieval records:

```sh
make build-search-chunks
```

The builder creates one chunk for each available verse translation and one for
each commentary passage. It preserves verse, commentary, and source links,
updates changed chunks without changing their IDs, and removes stale chunks
from the same builder version. It does not call an AI model or create
embeddings.

## Generate Embeddings

Set an OpenAI API key, then generate embeddings for chunks that are new or
whose content hash has changed:

```sh
export OPENAI_API_KEY='<your-api-key>'
make generate-embeddings
```

The generator uses `text-embedding-3-small` with 1,536 dimensions, sends
chunks in batches, and commits each completed batch so interrupted runs can be
resumed safely. If OpenAI applies a temporary rate limit, it waits for the
requested interval and retries automatically. Use
`go run ./cmd/generate-embeddings -limit 10` for a small end-to-end test, or
`make generate-embeddings EMBEDDING_BATCH_SIZE=16` to use smaller requests.

Look up Bhagavad Gita 2.47:

```sh
docker compose exec postgres psql -U narada -d narada -c \
  "SELECT c.number AS chapter, v.verse_number, v.original_text
   FROM verse v
   JOIN chapter c ON c.id = v.chapter_id
   WHERE c.number = 2 AND v.verse_number = '47';"
```

## Stop or Restart PostgreSQL

Stop the container while preserving its database volume:

```sh
docker compose stop postgres
```

Start it again:

```sh
docker compose up -d postgres
```

Do not remove the Docker volume unless the local database is intentionally being
discarded.

## Troubleshooting

If a command reports that `DATABASE_URL` is missing, export it again in the
current terminal:

```sh
export DATABASE_URL='postgres://narada:narada@localhost:5432/narada?sslmode=disable'
```

Check whether PostgreSQL is running and healthy:

```sh
docker compose ps
```

If port `5432` is already occupied, stop the conflicting PostgreSQL service or
change the host-side port in `compose.yaml` and update `DATABASE_URL` to match.

# Scripture API

Start the API after loading the database:

```sh
export DATABASE_URL='postgres://narada:narada@localhost:5432/narada?sslmode=disable'
go run ./cmd/api
```

The default base URL is `http://localhost:8080`. Set `PORT` to use another port.

## Endpoints

```text
GET /health
GET /api/v1/verses/random
GET /api/v1/search?q=:query
POST /api/v1/ask
GET /api/v1/scriptures/:scripture/chapters
GET /api/v1/scriptures/:scripture/chapters/:chapter
GET /api/v1/scriptures/:scripture/chapters/:chapter/verses
GET /api/v1/scriptures/:scripture/chapters/:chapter/verses/:verse
```

### Semantic search

Search translations and commentary by meaning rather than exact wording:

```sh
curl 'http://localhost:8080/api/v1/search?q=how+can+I+act+without+attachment&scripture=BG&limit=8'
```

The API embeds the question with the same model used for the stored chunks and
returns the nearest passages by cosine similarity. `limit` can be 1–20. Use
`kind=verse_translation` or `kind=commentary` to restrict the result type.
`OPENAI_API_KEY` must be present in the environment running the API.

### Ask for a grounded answer

Retrieve relevant passages and generate an answer constrained to that evidence:

```sh
curl -X POST http://localhost:8080/api/v1/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"How can I act without attachment?","scripture":"BG"}'
```

The response includes the generated answer, its token usage, and every source
passage supplied to the model. The answer model defaults to `gpt-5.6-luna` and
can be changed with `ANSWER_MODEL`. The model response is not stored by OpenAI
through the API request (`store` is false).

The currently loaded scripture identifier is `BG`. It is case-insensitive.

### Get a random verse

This endpoint takes no input and selects from every verse currently loaded:

```sh
curl http://localhost:8080/api/v1/verses/random
```

### List chapters

```sh
curl http://localhost:8080/api/v1/scriptures/BG/chapters
```

### Get one chapter

```sh
curl http://localhost:8080/api/v1/scriptures/BG/chapters/2
```

### List all verses in a chapter

```sh
curl http://localhost:8080/api/v1/scriptures/BG/chapters/2/verses
```

Verses are returned in their canonical sequence within the chapter.

### Get one verse

```sh
curl http://localhost:8080/api/v1/scriptures/BG/chapters/2/verses/47
```

The response includes the canonical reference, Sanskrit text, transliteration,
word meanings, chapter sequence, and scripture-wide sequence.

## Errors

Invalid chapter numbers return `400 Bad Request`. Unknown scriptures, chapters,
or verses return `404 Not Found`. Internal error details are not exposed in API
responses.

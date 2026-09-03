# Narada

Narada is a source-grounded Bhagavad Gita question-answering system. The current
implementation provides the PostgreSQL schema and validated corpus importer.

## Prerequisites

* Go 1.26 or later
* Node.js 22.13 or later
* Docker with Docker Compose

## Load the Gita locally

Start PostgreSQL:

```sh
docker compose up -d postgres
```

Set the database connection for the current shell:

```sh
export DATABASE_URL='postgres://narada:narada@localhost:5432/narada?sslmode=disable'
```

Apply migrations:

```sh
go run ./cmd/migrate
```

Validate and import all 18 chapters and 701 verses:

```sh
go run ./cmd/import-gita
```

Both migrations and corpus imports are safe to rerun. To validate the source
files without a running database:

```sh
go run ./cmd/import-gita -validate-only
```

See [docs/data-loading.md](docs/data-loading.md) for rerunning imports, recording
source versions, verifying database contents, and troubleshooting.

## Run the API

With PostgreSQL running, migrated, and loaded, start the HTTP API:

```sh
export DATABASE_URL='postgres://narada:narada@localhost:5432/narada?sslmode=disable'
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default. See
[docs/api.md](docs/api.md) for its chapter and verse endpoints.

## Run the web interface

For normal local development, start PostgreSQL, the API, and the web interface
together from the repository root:

```sh
make dev
```

Open `http://localhost:3000`. Press `Ctrl+C` to stop the API and web interface.
The PostgreSQL container remains available for the next run; stop it with
`make stop` when desired.

On a fresh database, load the Gita once before running the app:

```sh
make load
make dev
```

To run only the web interface while the API is already running:

```sh
cd web
npm run dev
```

Open `http://localhost:3000`. The interface is responsive across phone, tablet,
and desktop layouts. During local development, its `/api` requests are proxied
to `http://localhost:8080`. Set `API_URL` before `make dev` only when the Go API
runs elsewhere.

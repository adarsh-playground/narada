SHELL := /bin/bash

DATABASE_URL ?= postgres://narada:narada@localhost:5432/narada?sslmode=disable
PDF_PYTHON ?= python3
EMBEDDING_BATCH_SIZE ?= 32

.PHONY: dev load load-chinmayananda build-search-chunks generate-embeddings stop

dev:
	@docker compose up -d postgres
	@until docker compose exec -T postgres pg_isready -U narada -d narada >/dev/null 2>&1; do sleep 1; done
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate
	@mkdir -p .tmp
	@go build -o .tmp/narada-api ./cmd/api
	@set -eu; \
		DATABASE_URL='$(DATABASE_URL)' ./.tmp/narada-api & \
		api_pid=$$!; \
		trap 'kill $$api_pid 2>/dev/null || true' EXIT INT TERM; \
		sleep 1; \
		if ! kill -0 $$api_pid 2>/dev/null; then \
			echo 'Narada API could not start. Stop any previous make dev process and try again.' >&2; \
			exit 1; \
		fi; \
		cd web; \
		npm run dev

load:
	@docker compose up -d postgres
	@until docker compose exec -T postgres pg_isready -U narada -d narada >/dev/null 2>&1; do sleep 1; done
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/import-gita

load-chinmayananda:
	@docker compose up -d postgres
	@until docker compose exec -T postgres pg_isready -U narada -d narada >/dev/null 2>&1; do sleep 1; done
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate
	@$(PDF_PYTHON) data/gita/commentaries/Chinmayananda/parse_chapter.py --chapter all
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/import-chinmayananda

build-search-chunks:
	@docker compose up -d postgres
	@until docker compose exec -T postgres pg_isready -U narada -d narada >/dev/null 2>&1; do sleep 1; done
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/build-search-chunks

generate-embeddings:
	@docker compose up -d postgres
	@until docker compose exec -T postgres pg_isready -U narada -d narada >/dev/null 2>&1; do sleep 1; done
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate
	@DATABASE_URL='$(DATABASE_URL)' go run ./cmd/generate-embeddings -batch-size '$(EMBEDDING_BATCH_SIZE)'

stop:
	@docker compose stop postgres

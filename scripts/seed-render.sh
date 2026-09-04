#!/usr/bin/env bash
set -euo pipefail

echo "Loading Bhagavad Gita corpus"
./bin/import-gita

echo "Loading Swami Chinmayananda commentary"
./bin/import-chinmayananda

echo "Building semantic-search chunks"
./bin/build-search-chunks

echo "Generating missing embeddings"
./bin/generate-embeddings -batch-size "${EMBEDDING_BATCH_SIZE:-32}"

echo "Narada production data is ready"


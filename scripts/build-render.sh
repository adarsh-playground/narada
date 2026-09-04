#!/usr/bin/env bash
set -euo pipefail

mkdir -p bin

commands=(
  api
  migrate
  import-gita
  import-chinmayananda
  build-search-chunks
  generate-embeddings
)

for command in "${commands[@]}"; do
  go build -tags netgo -ldflags '-s -w' -o "bin/${command}" "./cmd/${command}"
done


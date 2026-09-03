package embedding

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type PendingChunk struct {
	ID            string
	Text          string
	ContentSHA256 string
}

func Generate(ctx context.Context, conn *pgx.Conn, provider Provider, batchSize, limit int) (int, error) {
	if batchSize < 1 || batchSize > 256 {
		return 0, fmt.Errorf("batch size must be between 1 and 256")
	}
	query := `SELECT sc.id, sc.text, sc.content_sha256 FROM search_chunk sc
		LEFT JOIN embedding e ON e.search_chunk_id=sc.id AND e.provider=$1 AND e.model=$2
		WHERE e.id IS NULL OR e.content_sha256<>sc.content_sha256 ORDER BY sc.kind, sc.citation_label, sc.id`
	args := []any{provider.Name(), provider.Model()}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
	}
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("load pending chunks: %w", err)
	}
	pending := make([]PendingChunk, 0)
	for rows.Next() {
		var chunk PendingChunk
		if err := rows.Scan(&chunk.ID, &chunk.Text, &chunk.ContentSHA256); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, chunk)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	completed := 0
	for start := 0; start < len(pending); start += batchSize {
		end := min(start+batchSize, len(pending))
		batch := pending[start:end]
		inputs := make([]string, len(batch))
		for i := range batch {
			inputs[i] = batch[i].Text
		}
		vectors, err := provider.Embed(ctx, inputs)
		if err != nil {
			return completed, fmt.Errorf("embed batch beginning with chunk %s: %w", batch[0].ID, err)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return completed, err
		}
		for i, chunk := range batch {
			if len(vectors[i]) != provider.Dimensions() {
				tx.Rollback(ctx)
				return completed, fmt.Errorf("chunk %s returned %d dimensions, want %d", chunk.ID, len(vectors[i]), provider.Dimensions())
			}
			if _, err := tx.Exec(ctx, `INSERT INTO embedding (search_chunk_id,provider,model,dimensions,content_sha256,embedding)
				VALUES ($1,$2,$3,$4,$5,$6::vector) ON CONFLICT (search_chunk_id,provider,model) DO UPDATE SET
				dimensions=EXCLUDED.dimensions,content_sha256=EXCLUDED.content_sha256,embedding=EXCLUDED.embedding,updated_at=now()`,
				chunk.ID, provider.Name(), provider.Model(), provider.Dimensions(), chunk.ContentSHA256, VectorLiteral(vectors[i])); err != nil {
				tx.Rollback(ctx)
				return completed, fmt.Errorf("store embedding for chunk %s: %w", chunk.ID, err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return completed, err
		}
		completed += len(batch)
	}
	return completed, nil
}

func VectorLiteral(vector []float32) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i, value := range vector {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	builder.WriteByte(']')
	return builder.String()
}

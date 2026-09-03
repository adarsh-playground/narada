package semanticsearch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/adarsh/narada/internal/embedding"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	KindTranslation = "verse_translation"
	KindCommentary  = "commentary"
)

type Result struct {
	ChunkID       string   `json:"chunk_id"`
	Kind          string   `json:"kind"`
	CitationLabel string   `json:"citation_label"`
	Source        string   `json:"source,omitempty"`
	Text          string   `json:"text"`
	VerseRefs     []string `json:"verse_references"`
	Similarity    float64  `json:"similarity"`
}

type Searcher interface {
	Search(ctx context.Context, query, scripture, kind string, limit int) ([]Result, error)
}

type Usage struct {
	EmbeddingModel       string `json:"embedding_model"`
	EmbeddingInputTokens int    `json:"embedding_input_tokens"`
}

type MeteredSearcher interface {
	Searcher
	SearchWithUsage(ctx context.Context, query, scripture, kind string, limit int) ([]Result, Usage, error)
}

type Service struct {
	pool     *pgxpool.Pool
	provider embedding.Provider
}

func New(pool *pgxpool.Pool, provider embedding.Provider) *Service {
	return &Service{pool: pool, provider: provider}
}

func (s *Service) Search(ctx context.Context, query, scripture, kind string, limit int) ([]Result, error) {
	results, _, err := s.SearchWithUsage(ctx, query, scripture, kind, limit)
	return results, err
}

func (s *Service) SearchWithUsage(ctx context.Context, query, scripture, kind string, limit int) ([]Result, Usage, error) {
	usage := Usage{EmbeddingModel: s.provider.Model()}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, usage, fmt.Errorf("search query cannot be empty")
	}
	if limit < 1 || limit > 20 {
		return nil, usage, fmt.Errorf("search limit must be between 1 and 20")
	}
	if kind != "" && kind != KindTranslation && kind != KindCommentary {
		return nil, usage, fmt.Errorf("unsupported search kind %q", kind)
	}

	var vectors [][]float32
	var err error
	if provider, ok := s.provider.(embedding.MeteredProvider); ok {
		vectors, usage.EmbeddingInputTokens, err = provider.EmbedWithUsage(ctx, []string{query})
	} else {
		vectors, err = s.provider.Embed(ctx, []string{query})
	}
	if err != nil {
		return nil, usage, fmt.Errorf("embed search query: %w", err)
	}
	if len(vectors) != 1 || len(vectors[0]) != s.provider.Dimensions() {
		return nil, usage, fmt.Errorf("search provider returned an invalid vector")
	}

	rows, err := s.pool.Query(ctx, `
		WITH nearest AS MATERIALIZED (
			SELECT sc.id, sc.kind, sc.citation_label, COALESCE(src.name, '') AS source,
			       sc.text, e.embedding <=> $1::vector AS distance
			FROM embedding e
			JOIN search_chunk sc ON sc.id=e.search_chunk_id
			JOIN scripture scr ON scr.id=sc.scripture_id
			LEFT JOIN source src ON src.id=sc.source_id
			WHERE e.provider=$2 AND e.model=$3 AND e.content_sha256=sc.content_sha256
			  AND ($4='' OR upper(scr.short_name)=upper($4))
			  AND ($5='' OR sc.kind=$5)
			ORDER BY e.embedding <=> $1::vector
			LIMIT $6
		)
		SELECT nearest.id, nearest.kind, nearest.citation_label, nearest.source, nearest.text,
		       COALESCE(array_agg(scr.short_name || ' ' || ch.number || '.' || v.verse_number
		           ORDER BY scv.position) FILTER (WHERE v.id IS NOT NULL), ARRAY[]::text[]),
		       1 - nearest.distance AS similarity
		FROM nearest
		LEFT JOIN search_chunk_verse scv ON scv.search_chunk_id=nearest.id
		LEFT JOIN verse v ON v.id=scv.verse_id
		LEFT JOIN chapter ch ON ch.id=v.chapter_id
		LEFT JOIN scripture scr ON scr.id=ch.scripture_id
		GROUP BY nearest.id, nearest.kind, nearest.citation_label, nearest.source,
		         nearest.text, nearest.distance
		ORDER BY nearest.distance`, embedding.VectorLiteral(vectors[0]), s.provider.Name(), s.provider.Model(), scripture, kind, limit)
	if err != nil {
		return nil, usage, fmt.Errorf("search embeddings: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0, limit)
	for rows.Next() {
		var result Result
		if err := rows.Scan(&result.ChunkID, &result.Kind, &result.CitationLabel, &result.Source,
			&result.Text, &result.VerseRefs, &result.Similarity); err != nil {
			return nil, usage, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, usage, fmt.Errorf("iterate search results: %w", err)
	}
	sortResults(results)
	return results, usage, nil
}

func sortResults(results []Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Kind != results[j].Kind {
			return results[i].Kind == KindCommentary
		}
		return results[i].Similarity > results[j].Similarity
	})
}

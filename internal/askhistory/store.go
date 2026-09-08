package askhistory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/adarsh/narada/internal/groundedanswer"
	"github.com/adarsh/narada/internal/semanticsearch"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EmbeddingPriceNanosPerToken int64 = 20
)

type Recorder interface {
	Start(ctx context.Context, question, scripture string) (string, error)
	Complete(ctx context.Context, id string, answer groundedanswer.Answer, usage semanticsearch.Usage, evidence []semanticsearch.Result, duration time.Duration) error
	Fail(ctx context.Context, id string, err error, usage semanticsearch.Usage, duration time.Duration) error
}

type CachedAnswer struct {
	Answer   groundedanswer.Answer
	Evidence []semanticsearch.Result
}

// CacheReader is optional so callers can continue using recorders that only
// persist request history.
type CacheReader interface {
	FindCompleted(ctx context.Context, question, scripture string) (CachedAnswer, bool, error)
}

type Interaction struct {
	ID                   string     `json:"id"`
	Scripture            string     `json:"scripture"`
	Question             string     `json:"question"`
	AnswerText           *string    `json:"answer_text,omitempty"`
	Status               string     `json:"status"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	EmbeddingModel       string     `json:"embedding_model"`
	AnswerModel          string     `json:"answer_model"`
	PromptVersion        string     `json:"prompt_version"`
	EmbeddingInputTokens int        `json:"embedding_input_tokens"`
	AnswerInputTokens    int        `json:"answer_input_tokens"`
	AnswerOutputTokens   int        `json:"answer_output_tokens"`
	TotalCostUSD         float64    `json:"total_cost_usd"`
	DurationMS           *int       `json:"duration_ms,omitempty"`
	Cached               bool       `json:"cached"`
	CreatedAt            time.Time  `json:"created_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
}

type InteractionSummary struct {
	TotalInteractions int     `json:"total_interactions"`
	Completed         int     `json:"completed"`
	Failed            int     `json:"failed"`
	Pending           int     `json:"pending"`
	Cached            int     `json:"cached"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
}

type InteractionPage struct {
	Interactions []Interaction      `json:"interactions"`
	Summary      InteractionSummary `json:"summary"`
	Total        int                `json:"total"`
	Limit        int                `json:"limit"`
	Offset       int                `json:"offset"`
}

type AdminReader interface {
	ListInteractions(ctx context.Context, limit, offset int, status string) (InteractionPage, error)
}

type Store struct {
	pool                           *pgxpool.Pool
	embeddingModel                 string
	answerModel                    string
	promptVersion                  string
	answerInputPriceNanosPerToken  int64
	answerOutputPriceNanosPerToken int64
}

func New(pool *pgxpool.Pool, embeddingModel, answerModel, promptVersion string) (*Store, error) {
	inputPrice, outputPrice, err := answerPrices(answerModel)
	if err != nil {
		return nil, err
	}
	if embeddingModel != "text-embedding-3-small" {
		return nil, fmt.Errorf("no configured price for embedding model %q", embeddingModel)
	}
	return &Store{pool: pool, embeddingModel: embeddingModel, answerModel: answerModel, promptVersion: promptVersion,
		answerInputPriceNanosPerToken: inputPrice, answerOutputPriceNanosPerToken: outputPrice}, nil
}

func (s *Store) Start(ctx context.Context, question, scripture string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `INSERT INTO ask_interaction
		(scripture_id,question,status,embedding_model,answer_model,prompt_version,
		 embedding_price_nanos_per_token,answer_input_price_nanos_per_token,answer_output_price_nanos_per_token)
		SELECT id,$2,'pending',$3,$4,$5,$6,$7,$8 FROM scripture WHERE upper(short_name)=upper($1) RETURNING id`,
		scripture, question, s.embeddingModel, s.answerModel, s.promptVersion, EmbeddingPriceNanosPerToken,
		s.answerInputPriceNanosPerToken, s.answerOutputPriceNanosPerToken).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("start ask history: %w", err)
	}
	return id, nil
}

func (s *Store) FindCompleted(ctx context.Context, question, scripture string) (CachedAnswer, bool, error) {
	const normalizedQuestion = `lower(regexp_replace(btrim(question), '\s+', ' ', 'g'))`
	var cached CachedAnswer
	var interactionID string
	err := s.pool.QueryRow(ctx, `SELECT ai.id, ai.answer_text, ai.answer_model
		FROM ask_interaction ai
		JOIN scripture scr ON scr.id=ai.scripture_id
		WHERE ai.status='completed' AND ai.answer_text IS NOT NULL
		  AND upper(scr.short_name)=upper($1::text)
		  AND `+normalizedQuestion+`=lower(regexp_replace(btrim($2::text), '\s+', ' ', 'g'))
		  AND ai.prompt_version=$3::text
		  AND (ai.answer_model=$4::text OR ai.answer_model LIKE $4::text || '-%')
		ORDER BY ai.completed_at DESC
		LIMIT 1`, scripture, question, s.promptVersion, s.answerModel).Scan(
		&interactionID, &cached.Answer.Text, &cached.Answer.Model)
	if err != nil {
		if err == pgx.ErrNoRows {
			return CachedAnswer{}, false, nil
		}
		return CachedAnswer{}, false, fmt.Errorf("find cached answer: %w", err)
	}

	rows, err := s.pool.Query(ctx, `SELECT COALESCE(search_chunk_id::text,''),kind,citation_label,COALESCE(source_name,''),
		text_snapshot,verse_references,similarity
		FROM ask_interaction_evidence WHERE ask_interaction_id=$1 ORDER BY rank`, interactionID)
	if err != nil {
		return CachedAnswer{}, false, fmt.Errorf("load cached answer evidence: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result semanticsearch.Result
		var refs []byte
		if err := rows.Scan(&result.ChunkID, &result.Kind, &result.CitationLabel, &result.Source,
			&result.Text, &refs, &result.Similarity); err != nil {
			return CachedAnswer{}, false, fmt.Errorf("scan cached answer evidence: %w", err)
		}
		if err := json.Unmarshal(refs, &result.VerseRefs); err != nil {
			return CachedAnswer{}, false, fmt.Errorf("decode cached answer evidence: %w", err)
		}
		cached.Evidence = append(cached.Evidence, result)
	}
	if err := rows.Err(); err != nil {
		return CachedAnswer{}, false, fmt.Errorf("iterate cached answer evidence: %w", err)
	}
	return cached, true, nil
}

func (s *Store) ListInteractions(ctx context.Context, limit, offset int, status string) (InteractionPage, error) {
	page := InteractionPage{Interactions: []Interaction{}, Limit: limit, Offset: offset}
	err := s.pool.QueryRow(ctx, `SELECT count(*)::integer,
		count(*) FILTER (WHERE status='completed')::integer,
		count(*) FILTER (WHERE status='failed')::integer,
		count(*) FILTER (WHERE status='pending')::integer,
		count(*) FILTER (WHERE status='completed' AND answer_text IS NOT NULL
			AND embedding_input_tokens=0 AND answer_input_tokens=0 AND answer_output_tokens=0)::integer,
		COALESCE(sum(embedding_input_tokens+answer_input_tokens+answer_output_tokens),0)::bigint,
		COALESCE(sum(total_cost_usd),0)::float8
		FROM ask_interaction`).Scan(&page.Summary.TotalInteractions, &page.Summary.Completed,
		&page.Summary.Failed, &page.Summary.Pending, &page.Summary.Cached,
		&page.Summary.TotalTokens, &page.Summary.TotalCostUSD)
	if err != nil {
		return InteractionPage{}, fmt.Errorf("summarize ask history: %w", err)
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*)::integer FROM ask_interaction
		WHERE ($1::text='' OR status=$1::text)`, status).Scan(&page.Total)
	if err != nil {
		return InteractionPage{}, fmt.Errorf("count ask history: %w", err)
	}

	rows, err := s.pool.Query(ctx, `SELECT ai.id,COALESCE(scr.short_name,''),ai.question,ai.answer_text,ai.status,ai.error_message,
		ai.embedding_model,ai.answer_model,ai.prompt_version,ai.embedding_input_tokens,
		ai.answer_input_tokens,ai.answer_output_tokens,ai.total_cost_usd::float8,ai.duration_ms,
		(ai.status='completed' AND ai.answer_text IS NOT NULL AND ai.embedding_input_tokens=0
			AND ai.answer_input_tokens=0 AND ai.answer_output_tokens=0) AS cached,
		ai.created_at,ai.completed_at
		FROM ask_interaction ai
		LEFT JOIN scripture scr ON scr.id=ai.scripture_id
		WHERE ($1::text='' OR ai.status=$1::text)
		ORDER BY ai.created_at DESC
		LIMIT $2::integer OFFSET $3::integer`, status, limit, offset)
	if err != nil {
		return InteractionPage{}, fmt.Errorf("list ask history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var interaction Interaction
		if err := rows.Scan(&interaction.ID, &interaction.Scripture, &interaction.Question,
			&interaction.AnswerText, &interaction.Status, &interaction.ErrorMessage,
			&interaction.EmbeddingModel, &interaction.AnswerModel, &interaction.PromptVersion,
			&interaction.EmbeddingInputTokens, &interaction.AnswerInputTokens,
			&interaction.AnswerOutputTokens, &interaction.TotalCostUSD, &interaction.DurationMS,
			&interaction.Cached, &interaction.CreatedAt, &interaction.CompletedAt); err != nil {
			return InteractionPage{}, fmt.Errorf("scan ask history: %w", err)
		}
		page.Interactions = append(page.Interactions, interaction)
	}
	if err := rows.Err(); err != nil {
		return InteractionPage{}, fmt.Errorf("iterate ask history: %w", err)
	}
	return page, nil
}

func answerPrices(model string) (int64, int64, error) {
	switch {
	case strings.HasPrefix(model, "gpt-5.6-luna"):
		return 200, 1200, nil
	case strings.HasPrefix(model, "gpt-5.6-terra"):
		return 2000, 12000, nil
	case model == "gpt-5.6" || strings.HasPrefix(model, "gpt-5.6-sol"):
		return 4000, 20000, nil
	default:
		return 0, 0, fmt.Errorf("no configured price for answer model %q", model)
	}
}

func (s *Store) Complete(ctx context.Context, id string, answer groundedanswer.Answer, usage semanticsearch.Usage, evidence []semanticsearch.Result, duration time.Duration) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE ask_interaction SET status='completed',answer_text=$2,
		embedding_model=COALESCE(NULLIF($3::text,''),embedding_model),
		answer_model=COALESCE(NULLIF($4::text,''),answer_model),
		embedding_input_tokens=$5::integer,answer_input_tokens=$6::integer,answer_output_tokens=$7::integer,
		embedding_cost_nanos=$5::integer*embedding_price_nanos_per_token,
		answer_input_cost_nanos=$6::integer*answer_input_price_nanos_per_token,
		answer_output_cost_nanos=$7::integer*answer_output_price_nanos_per_token,
		duration_ms=$8,completed_at=now() WHERE id=$1`, id, answer.Text, usage.EmbeddingModel, answer.Model,
		usage.EmbeddingInputTokens, answer.InputTokens, answer.OutputTokens, duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("complete ask history: %w", err)
	}
	for rank, result := range evidence {
		refs, err := json.Marshal(result.VerseRefs)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO ask_interaction_evidence
			(ask_interaction_id,rank,search_chunk_id,kind,citation_label,source_name,text_snapshot,verse_references,similarity)
			VALUES ($1,$2,NULLIF($3::text,'')::uuid,$4,$5,$6,$7,$8,$9)`, id, rank+1, result.ChunkID, result.Kind,
			result.CitationLabel, result.Source, result.Text, refs, result.Similarity)
		if err != nil {
			return fmt.Errorf("store ask evidence: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Fail(ctx context.Context, id string, failure error, usage semanticsearch.Usage, duration time.Duration) error {
	message := failure.Error()
	if len(message) > 2000 {
		message = message[:2000]
	}
	_, err := s.pool.Exec(ctx, `UPDATE ask_interaction SET status='failed',error_message=$2,embedding_model=$3,
		embedding_input_tokens=$4::integer,embedding_cost_nanos=$4::integer*embedding_price_nanos_per_token,
		duration_ms=$5,completed_at=now() WHERE id=$1`, id, message, usage.EmbeddingModel,
		usage.EmbeddingInputTokens, duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("fail ask history: %w", err)
	}
	return nil
}

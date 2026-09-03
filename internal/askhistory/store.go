package askhistory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/adarsh/narada/internal/groundedanswer"
	"github.com/adarsh/narada/internal/semanticsearch"
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
	_, err = tx.Exec(ctx, `UPDATE ask_interaction SET status='completed',answer_text=$2,embedding_model=$3,answer_model=$4,
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
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, rank+1, result.ChunkID, result.Kind,
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

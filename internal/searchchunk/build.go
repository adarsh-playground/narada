package searchchunk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const BuilderVersion = "1"

type Chunk struct {
	Kind          string
	StableKey     string
	ScriptureID   string
	SourceID      string
	CitationLabel string
	Text          string
	VerseIDs      []string
	CommentaryID  string
	ContentSHA256 string
}

func VerseTranslationChunk(scriptureID, verseID, sourceID, scriptureName, reference, sourceName, translation string) Chunk {
	text := fmt.Sprintf("%s %s\nTranslation by %s\n\n%s", strings.TrimSpace(scriptureName), strings.TrimPrefix(reference, "BG "), strings.TrimSpace(sourceName), strings.TrimSpace(translation))
	return newChunk("verse_translation", "translation:"+verseID+":"+sourceID, scriptureID, sourceID, reference, text, []string{verseID}, "")
}

func CommentaryChunk(scriptureID, commentaryID, sourceID, scriptureName, citationLabel, sourceName, commentary string, verseIDs []string) Chunk {
	text := fmt.Sprintf("%s %s\nCommentary by %s\n\n%s", strings.TrimSpace(scriptureName), strings.TrimPrefix(citationLabel, "BG "), strings.TrimSpace(sourceName), strings.TrimSpace(commentary))
	return newChunk("commentary", "commentary:"+commentaryID, scriptureID, sourceID, citationLabel, text, verseIDs, commentaryID)
}

func newChunk(kind, stableKey, scriptureID, sourceID, citationLabel, text string, verseIDs []string, commentaryID string) Chunk {
	hash := sha256.Sum256([]byte(text))
	return Chunk{Kind: kind, StableKey: stableKey, ScriptureID: scriptureID, SourceID: sourceID, CitationLabel: citationLabel, Text: text, VerseIDs: verseIDs, CommentaryID: commentaryID, ContentSHA256: fmt.Sprintf("%x", hash)}
}

func Build(ctx context.Context, conn *pgx.Conn, shortName string) (int, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin chunk build: %w", err)
	}
	defer tx.Rollback(ctx)

	var scriptureID, scriptureName, buildToken string
	if err := tx.QueryRow(ctx, `SELECT id, name, gen_random_uuid() FROM scripture WHERE upper(short_name)=upper($1)`, shortName).Scan(&scriptureID, &scriptureName, &buildToken); err != nil {
		return 0, fmt.Errorf("find scripture %s: %w", shortName, err)
	}

	chunks, err := loadChunks(ctx, tx, scriptureID, scriptureName)
	if err != nil {
		return 0, err
	}
	if len(chunks) == 0 {
		return 0, fmt.Errorf("no source material found for %s", shortName)
	}

	for _, chunk := range chunks {
		var chunkID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO search_chunk (scripture_id,source_id,kind,stable_key,citation_label,text,content_sha256,builder_version,build_token)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (scripture_id,stable_key) DO UPDATE SET
				source_id=EXCLUDED.source_id, kind=EXCLUDED.kind, citation_label=EXCLUDED.citation_label,
				text=EXCLUDED.text, content_sha256=EXCLUDED.content_sha256,
				builder_version=EXCLUDED.builder_version, build_token=EXCLUDED.build_token, updated_at=now()
			RETURNING id`, chunk.ScriptureID, chunk.SourceID, chunk.Kind, chunk.StableKey, chunk.CitationLabel, chunk.Text, chunk.ContentSHA256, BuilderVersion, buildToken).Scan(&chunkID); err != nil {
			return 0, fmt.Errorf("upsert chunk %s: %w", chunk.StableKey, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM search_chunk_verse WHERE search_chunk_id=$1`, chunkID); err != nil {
			return 0, err
		}
		for position, verseID := range chunk.VerseIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO search_chunk_verse (search_chunk_id,verse_id,position) VALUES ($1,$2,$3)`, chunkID, verseID, position+1); err != nil {
				return 0, fmt.Errorf("link chunk verse: %w", err)
			}
		}
		if chunk.CommentaryID != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO search_chunk_commentary (search_chunk_id,commentary_id) VALUES ($1,$2) ON CONFLICT (search_chunk_id) DO UPDATE SET commentary_id=EXCLUDED.commentary_id`, chunkID, chunk.CommentaryID); err != nil {
				return 0, fmt.Errorf("link commentary chunk: %w", err)
			}
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM search_chunk WHERE scripture_id=$1 AND builder_version=$2 AND build_token<>$3`, scriptureID, BuilderVersion, buildToken); err != nil {
		return 0, fmt.Errorf("remove stale chunks: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO search_chunk_build (scripture_id,builder_version,chunk_count) VALUES ($1,$2,$3)`, scriptureID, BuilderVersion, len(chunks)); err != nil {
		return 0, fmt.Errorf("record chunk build: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit chunk build: %w", err)
	}
	return len(chunks), nil
}

func loadChunks(ctx context.Context, tx pgx.Tx, scriptureID, scriptureName string) ([]Chunk, error) {
	chunks := make([]Chunk, 0, 1400)
	rows, err := tx.Query(ctx, `
		SELECT v.id, t.source_id, s.short_name || ' ' || c.number || '.' || v.verse_number, src.name, t.text
		FROM translation t JOIN verse v ON v.id=t.verse_id JOIN chapter c ON c.id=v.chapter_id
		JOIN scripture s ON s.id=c.scripture_id JOIN source src ON src.id=t.source_id
		WHERE s.id=$1 ORDER BY v.scripture_sequence_number, src.name`, scriptureID)
	if err != nil {
		return nil, fmt.Errorf("load translations: %w", err)
	}
	for rows.Next() {
		var verseID, sourceID, reference, sourceName, text string
		if err := rows.Scan(&verseID, &sourceID, &reference, &sourceName, &text); err != nil {
			rows.Close()
			return nil, err
		}
		chunks = append(chunks, VerseTranslationChunk(scriptureID, verseID, sourceID, scriptureName, reference, sourceName, text))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = tx.Query(ctx, `
		SELECT cm.id, cm.source_id, COALESCE(cm.citation_label, s.short_name || ' ' || c.number), src.name, cm.text,
		       array_agg(v.id::text ORDER BY cv.position)
		FROM commentary cm JOIN chapter c ON c.id=cm.chapter_id JOIN scripture s ON s.id=c.scripture_id
		JOIN source src ON src.id=cm.source_id JOIN commentary_verse cv ON cv.commentary_id=cm.id JOIN verse v ON v.id=cv.verse_id
		WHERE s.id=$1 GROUP BY cm.id, s.short_name, c.number, src.name ORDER BY c.number, cm.sequence_number`, scriptureID)
	if err != nil {
		return nil, fmt.Errorf("load commentary: %w", err)
	}
	for rows.Next() {
		var commentaryID, sourceID, citationLabel, sourceName, text string
		var verseIDs []string
		if err := rows.Scan(&commentaryID, &sourceID, &citationLabel, &sourceName, &text, &verseIDs); err != nil {
			rows.Close()
			return nil, err
		}
		chunks = append(chunks, CommentaryChunk(scriptureID, commentaryID, sourceID, scriptureName, citationLabel, sourceName, text, verseIDs))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	return chunks, nil
}

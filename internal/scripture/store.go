package scripture

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("scripture record not found")

type Store interface {
	ListChapters(ctx context.Context, scripture string) ([]Chapter, error)
	GetChapter(ctx context.Context, scripture string, chapterNumber int) (Chapter, error)
	ListVerses(ctx context.Context, scripture string, chapterNumber int) ([]Verse, error)
	GetVerse(ctx context.Context, scripture string, chapterNumber int, verseNumber string) (Verse, error)
	RandomVerse(ctx context.Context) (Verse, error)
}

type PostgresStore struct {
	conn queryer
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewPostgresStore(conn queryer) *PostgresStore {
	return &PostgresStore{conn: conn}
}

func (s *PostgresStore) ListChapters(ctx context.Context, shortName string) ([]Chapter, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT c.id, c.number, COALESCE(c.title, ''), COALESCE(c.original_title, ''),
		       COALESCE(c.transliterated_title, ''), COALESCE(c.meaning, ''),
		       COALESCE(c.summary, ''), COALESCE(c.summary_hindi, ''), COUNT(v.id)::int
		FROM scripture s
		JOIN chapter c ON c.scripture_id = s.id
		LEFT JOIN verse v ON v.chapter_id = c.id
		WHERE upper(s.short_name) = upper($1)
		GROUP BY c.id
		ORDER BY c.number
	`, shortName)
	if err != nil {
		return nil, fmt.Errorf("list chapters: %w", err)
	}
	defer rows.Close()

	chapters := make([]Chapter, 0)
	for rows.Next() {
		var chapter Chapter
		if err := rows.Scan(&chapter.ID, &chapter.Number, &chapter.Title, &chapter.OriginalTitle,
			&chapter.TransliteratedTitle, &chapter.Meaning, &chapter.Summary,
			&chapter.SummaryHindi, &chapter.VerseCount); err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}
		chapters = append(chapters, chapter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters: %w", err)
	}
	if len(chapters) == 0 {
		return nil, ErrNotFound
	}
	return chapters, nil
}

func (s *PostgresStore) GetChapter(ctx context.Context, shortName string, chapterNumber int) (Chapter, error) {
	var chapter Chapter
	err := s.conn.QueryRow(ctx, `
		SELECT c.id, c.number, COALESCE(c.title, ''), COALESCE(c.original_title, ''),
		       COALESCE(c.transliterated_title, ''), COALESCE(c.meaning, ''),
		       COALESCE(c.summary, ''), COALESCE(c.summary_hindi, ''), COUNT(v.id)::int
		FROM scripture s
		JOIN chapter c ON c.scripture_id = s.id
		LEFT JOIN verse v ON v.chapter_id = c.id
		WHERE upper(s.short_name) = upper($1) AND c.number = $2
		GROUP BY c.id
	`, shortName, chapterNumber).Scan(&chapter.ID, &chapter.Number, &chapter.Title,
		&chapter.OriginalTitle, &chapter.TransliteratedTitle, &chapter.Meaning,
		&chapter.Summary, &chapter.SummaryHindi, &chapter.VerseCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Chapter{}, ErrNotFound
	}
	if err != nil {
		return Chapter{}, fmt.Errorf("get chapter: %w", err)
	}
	return chapter, nil
}

func (s *PostgresStore) ListVerses(ctx context.Context, shortName string, chapterNumber int) ([]Verse, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT v.id, s.short_name || ' ' || c.number || '.' || v.verse_number,
		       c.number, v.verse_number, v.sequence_number, v.scripture_sequence_number,
		       v.original_text, COALESCE(v.transliteration, ''), COALESCE(v.word_meanings, ''),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'text', t.text) ORDER BY src.name)
		                 FROM translation t JOIN source src ON src.id=t.source_id WHERE t.verse_id=v.id), '[]'::jsonb),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'citation_label', COALESCE(cm.citation_label, ''), 'text', cm.text) ORDER BY cm.sequence_number)
		                 FROM commentary_verse cv JOIN commentary cm ON cm.id=cv.commentary_id JOIN source src ON src.id=cm.source_id WHERE cv.verse_id=v.id), '[]'::jsonb)
		FROM scripture s
		JOIN chapter c ON c.scripture_id = s.id
		JOIN verse v ON v.chapter_id = c.id
		WHERE upper(s.short_name) = upper($1) AND c.number = $2
		ORDER BY v.sequence_number
	`, shortName, chapterNumber)
	if err != nil {
		return nil, fmt.Errorf("list verses: %w", err)
	}
	defer rows.Close()

	verses := make([]Verse, 0)
	for rows.Next() {
		verse, err := scanVerse(rows)
		if err != nil {
			return nil, err
		}
		verses = append(verses, verse)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verses: %w", err)
	}
	if len(verses) == 0 {
		return nil, ErrNotFound
	}
	return verses, nil
}

func (s *PostgresStore) GetVerse(ctx context.Context, shortName string, chapterNumber int, verseNumber string) (Verse, error) {
	row := s.conn.QueryRow(ctx, `
		SELECT v.id, s.short_name || ' ' || c.number || '.' || v.verse_number,
		       c.number, v.verse_number, v.sequence_number, v.scripture_sequence_number,
		       v.original_text, COALESCE(v.transliteration, ''), COALESCE(v.word_meanings, ''),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'text', t.text) ORDER BY src.name)
		                 FROM translation t JOIN source src ON src.id=t.source_id WHERE t.verse_id=v.id), '[]'::jsonb),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'citation_label', COALESCE(cm.citation_label, ''), 'text', cm.text) ORDER BY cm.sequence_number)
		                 FROM commentary_verse cv JOIN commentary cm ON cm.id=cv.commentary_id JOIN source src ON src.id=cm.source_id WHERE cv.verse_id=v.id), '[]'::jsonb)
		FROM scripture s
		JOIN chapter c ON c.scripture_id = s.id
		JOIN verse v ON v.chapter_id = c.id
		WHERE upper(s.short_name) = upper($1) AND c.number = $2 AND v.verse_number = $3
	`, shortName, chapterNumber, verseNumber)
	verse, err := scanVerse(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Verse{}, ErrNotFound
	}
	if err != nil {
		return Verse{}, err
	}
	return verse, nil
}

func (s *PostgresStore) RandomVerse(ctx context.Context) (Verse, error) {
	row := s.conn.QueryRow(ctx, `
		SELECT v.id, s.short_name || ' ' || c.number || '.' || v.verse_number,
		       c.number, v.verse_number, v.sequence_number, v.scripture_sequence_number,
		       v.original_text, COALESCE(v.transliteration, ''), COALESCE(v.word_meanings, ''),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'text', t.text) ORDER BY src.name)
		                 FROM translation t JOIN source src ON src.id=t.source_id WHERE t.verse_id=v.id), '[]'::jsonb),
		       COALESCE((SELECT jsonb_agg(jsonb_build_object('source', src.name, 'citation_label', COALESCE(cm.citation_label, ''), 'text', cm.text) ORDER BY cm.sequence_number)
		                 FROM commentary_verse cv JOIN commentary cm ON cm.id=cv.commentary_id JOIN source src ON src.id=cm.source_id WHERE cv.verse_id=v.id), '[]'::jsonb)
		FROM verse v
		JOIN chapter c ON c.id = v.chapter_id
		JOIN scripture s ON s.id = c.scripture_id
		ORDER BY random()
		LIMIT 1
	`)
	verse, err := scanVerse(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Verse{}, ErrNotFound
	}
	if err != nil {
		return Verse{}, err
	}
	return verse, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVerse(row rowScanner) (Verse, error) {
	var verse Verse
	if err := row.Scan(&verse.ID, &verse.Reference, &verse.ChapterNumber,
		&verse.VerseNumber, &verse.SequenceNumber, &verse.ScriptureSequenceNumber,
		&verse.OriginalText, &verse.Transliteration, &verse.WordMeanings,
		&verse.Translations, &verse.Commentaries); err != nil {
		return Verse{}, fmt.Errorf("scan verse: %w", err)
	}
	return verse, nil
}

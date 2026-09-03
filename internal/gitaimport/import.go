package gitaimport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const ImporterVersion = "1"

type Chapter struct {
	ID                  int    `json:"id"`
	ChapterNumber       int    `json:"chapter_number"`
	Name                string `json:"name"`
	NameTranslation     string `json:"name_translation"`
	NameTransliterated  string `json:"name_transliterated"`
	NameMeaning         string `json:"name_meaning"`
	ChapterSummary      string `json:"chapter_summary"`
	ChapterSummaryHindi string `json:"chapter_summary_hindi"`
	VersesCount         int    `json:"verses_count"`
}

type Verse struct {
	ID              int    `json:"id"`
	ExternalID      int    `json:"externalId"`
	ChapterNumber   int    `json:"chapter_number"`
	VerseNumber     int    `json:"verse_number"`
	VerseOrder      int    `json:"verse_order"`
	Text            string `json:"text"`
	Transliteration string `json:"transliteration"`
	WordMeanings    string `json:"word_meanings"`
}

type Corpus struct {
	Chapters []Chapter
	Verses   []Verse
}

func Load(chaptersPath, versesPath string) (Corpus, error) {
	var corpus Corpus
	if err := readJSON(chaptersPath, &corpus.Chapters); err != nil {
		return Corpus{}, err
	}
	if err := readJSON(versesPath, &corpus.Verses); err != nil {
		return Corpus{}, err
	}
	if err := corpus.Validate(); err != nil {
		return Corpus{}, err
	}
	return corpus, nil
}

func readJSON(path string, target any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(contents, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c Corpus) Validate() error {
	if len(c.Chapters) != 18 {
		return fmt.Errorf("expected 18 chapters, found %d", len(c.Chapters))
	}
	if len(c.Verses) != 701 {
		return fmt.Errorf("expected 701 verses, found %d", len(c.Verses))
	}

	expected := make(map[int]int, len(c.Chapters))
	seenChapters := make(map[int]bool, len(c.Chapters))
	for _, chapter := range c.Chapters {
		if chapter.ChapterNumber < 1 || chapter.ChapterNumber > 18 || seenChapters[chapter.ChapterNumber] {
			return fmt.Errorf("invalid or duplicate chapter number %d", chapter.ChapterNumber)
		}
		seenChapters[chapter.ChapterNumber] = true
		expected[chapter.ChapterNumber] = chapter.VersesCount
	}

	counts := make(map[int]int, len(c.Chapters))
	references := make(map[string]bool, len(c.Verses))
	orders := make(map[int]bool, len(c.Verses))
	for _, verse := range c.Verses {
		if !seenChapters[verse.ChapterNumber] {
			return fmt.Errorf("verse %d refers to unknown chapter %d", verse.ID, verse.ChapterNumber)
		}
		ref := fmt.Sprintf("%d.%d", verse.ChapterNumber, verse.VerseNumber)
		if references[ref] {
			return fmt.Errorf("duplicate verse reference %s", ref)
		}
		if orders[verse.VerseOrder] {
			return fmt.Errorf("duplicate scripture verse order %d", verse.VerseOrder)
		}
		if strings.TrimSpace(verse.Text) == "" || strings.TrimSpace(verse.Transliteration) == "" {
			return fmt.Errorf("verse %s has empty Sanskrit or transliteration", ref)
		}
		references[ref] = true
		orders[verse.VerseOrder] = true
		counts[verse.ChapterNumber]++
	}
	for chapter, count := range expected {
		if counts[chapter] != count {
			return fmt.Errorf("chapter %d declares %d verses but contains %d", chapter, count, counts[chapter])
		}
	}
	return nil
}

func (c Corpus) Import(ctx context.Context, conn *pgx.Conn, sourceVersion string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)

	var scriptureID string
	err = tx.QueryRow(ctx, `
		INSERT INTO scripture (name, short_name, original_language, description)
		VALUES ('Bhagavad Gita', 'BG', 'Sanskrit', 'The dialogue between Krishna and Arjuna within the Mahabharata.')
		ON CONFLICT (short_name) DO UPDATE SET
			name = EXCLUDED.name,
			original_language = EXCLUDED.original_language,
			description = EXCLUDED.description
		RETURNING id
	`).Scan(&scriptureID)
	if err != nil {
		return fmt.Errorf("upsert scripture: %w", err)
	}

	chapterIDs := make(map[int]string, len(c.Chapters))
	for _, chapter := range c.Chapters {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO chapter (
				scripture_id, number, source_external_id, title, original_title,
				transliterated_title, meaning, summary, summary_hindi, expected_verse_count
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (scripture_id, number) DO UPDATE SET
				source_external_id = EXCLUDED.source_external_id,
				title = EXCLUDED.title,
				original_title = EXCLUDED.original_title,
				transliterated_title = EXCLUDED.transliterated_title,
				meaning = EXCLUDED.meaning,
				summary = EXCLUDED.summary,
				summary_hindi = EXCLUDED.summary_hindi,
				expected_verse_count = EXCLUDED.expected_verse_count
			RETURNING id
		`, scriptureID, chapter.ChapterNumber, strconv.Itoa(chapter.ID), chapter.NameTranslation,
			chapter.Name, chapter.NameTransliterated, chapter.NameMeaning, chapter.ChapterSummary,
			chapter.ChapterSummaryHindi, chapter.VersesCount).Scan(&id)
		if err != nil {
			return fmt.Errorf("upsert chapter %d: %w", chapter.ChapterNumber, err)
		}
		chapterIDs[chapter.ChapterNumber] = id
	}

	for _, verse := range c.Verses {
		_, err := tx.Exec(ctx, `
			INSERT INTO verse (
				chapter_id, verse_number, sequence_number, scripture_sequence_number,
				source_external_id, original_text, transliteration, word_meanings
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (chapter_id, verse_number) DO UPDATE SET
				sequence_number = EXCLUDED.sequence_number,
				scripture_sequence_number = EXCLUDED.scripture_sequence_number,
				source_external_id = EXCLUDED.source_external_id,
				original_text = EXCLUDED.original_text,
				transliteration = EXCLUDED.transliteration,
				word_meanings = EXCLUDED.word_meanings
		`, chapterIDs[verse.ChapterNumber], strconv.Itoa(verse.VerseNumber), verse.VerseNumber,
			verse.VerseOrder, strconv.Itoa(verse.ExternalID), strings.TrimSpace(verse.Text),
			strings.TrimSpace(verse.Transliteration), strings.TrimSpace(verse.WordMeanings))
		if err != nil {
			return fmt.Errorf("upsert verse %d.%d: %w", verse.ChapterNumber, verse.VerseNumber, err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO data_import (
			dataset, source_url, source_version, license, importer_version, chapter_count, verse_count
		) VALUES ('gita/gita', 'https://github.com/gita/gita', $1, 'Unlicense', $2, $3, $4)
	`, sourceVersion, ImporterVersion, len(c.Chapters), len(c.Verses)); err != nil {
		return fmt.Errorf("record import: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}

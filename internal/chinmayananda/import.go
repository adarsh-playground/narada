package chinmayananda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const ImporterVersion = "1"

type Source struct {
	Name        string `json:"name"`
	Publication string `json:"publication"`
	Language    string `json:"language"`
}

type Translation struct {
	VerseNumber string `json:"verse_number"`
	Text        string `json:"text"`
}

type CommentaryPassage struct {
	SequenceNumber int      `json:"sequence_number"`
	VerseNumbers   []string `json:"verse_numbers"`
	CitationLabel  string   `json:"citation_label"`
	Text           string   `json:"text"`
}

type Chapter struct {
	Chapter            int                 `json:"chapter"`
	Source             Source              `json:"source"`
	Translations       []Translation       `json:"translations"`
	CommentaryPassages []CommentaryPassage `json:"commentary_passages"`
}

func Load(path string) (Chapter, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Chapter{}, fmt.Errorf("read %s: %w", path, err)
	}
	var chapter Chapter
	if err := json.Unmarshal(contents, &chapter); err != nil {
		return Chapter{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := chapter.Validate(); err != nil {
		return Chapter{}, err
	}
	return chapter, nil
}

func (c Chapter) Validate() error {
	if c.Chapter < 1 || strings.TrimSpace(c.Source.Name) == "" || strings.TrimSpace(c.Source.Language) == "" {
		return fmt.Errorf("invalid chapter or source metadata")
	}
	seen := map[string]bool{}
	for _, translation := range c.Translations {
		if _, err := strconv.Atoi(translation.VerseNumber); err != nil || seen[translation.VerseNumber] || strings.TrimSpace(translation.Text) == "" {
			return fmt.Errorf("invalid or duplicate translation for verse %q", translation.VerseNumber)
		}
		seen[translation.VerseNumber] = true
	}
	for i, passage := range c.CommentaryPassages {
		if passage.SequenceNumber != i+1 || len(passage.VerseNumbers) == 0 || strings.TrimSpace(passage.Text) == "" {
			return fmt.Errorf("invalid commentary passage at position %d", i+1)
		}
		for _, verse := range passage.VerseNumbers {
			if !seen[verse] {
				return fmt.Errorf("commentary refers to verse %s without a translation", verse)
			}
		}
	}
	return nil
}

func (c Chapter) Import(ctx context.Context, conn *pgx.Conn, sourcePath string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)

	var chapterID string
	if err := tx.QueryRow(ctx, `SELECT c.id FROM chapter c JOIN scripture s ON s.id=c.scripture_id WHERE s.short_name='BG' AND c.number=$1`, c.Chapter).Scan(&chapterID); err != nil {
		return fmt.Errorf("find BG chapter %d (run import-gita first): %w", c.Chapter, err)
	}
	verseIDs := map[string]string{}
	rows, err := tx.Query(ctx, `SELECT verse_number, id FROM verse WHERE chapter_id=$1`, chapterID)
	if err != nil {
		return fmt.Errorf("load verses: %w", err)
	}
	for rows.Next() {
		var number, id string
		if err := rows.Scan(&number, &id); err != nil {
			rows.Close()
			return err
		}
		verseIDs[number] = id
	}
	rows.Close()

	var translationSourceID, commentarySourceID string
	upsertSource := func(kind string, target *string) error {
		return tx.QueryRow(ctx, `INSERT INTO source (name,type,tradition,language,publication,description)
			VALUES ($1,$2,'Advaita Vedanta',$3,$4,'Translation and commentary extracted from The Holy Geeta.')
			ON CONFLICT (name,type) DO UPDATE SET language=EXCLUDED.language, publication=EXCLUDED.publication RETURNING id`,
			c.Source.Name, kind, c.Source.Language, c.Source.Publication).Scan(target)
	}
	if err := upsertSource("translation", &translationSourceID); err != nil {
		return fmt.Errorf("upsert translation source: %w", err)
	}
	if err := upsertSource("commentary", &commentarySourceID); err != nil {
		return fmt.Errorf("upsert commentary source: %w", err)
	}

	for _, item := range c.Translations {
		verseID, ok := verseIDs[item.VerseNumber]
		if !ok {
			return fmt.Errorf("BG %d.%s not found", c.Chapter, item.VerseNumber)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO translation (verse_id,source_id,text) VALUES ($1,$2,$3)
			ON CONFLICT (verse_id,source_id) DO UPDATE SET text=EXCLUDED.text`, verseID, translationSourceID, strings.TrimSpace(item.Text)); err != nil {
			return fmt.Errorf("upsert translation %d.%s: %w", c.Chapter, item.VerseNumber, err)
		}
	}
	for _, passage := range c.CommentaryPassages {
		var commentaryID string
		if err := tx.QueryRow(ctx, `INSERT INTO commentary (source_id,chapter_id,text,sequence_number,citation_label) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_id,chapter_id,sequence_number) DO UPDATE SET text=EXCLUDED.text,citation_label=EXCLUDED.citation_label RETURNING id`,
			commentarySourceID, chapterID, strings.TrimSpace(passage.Text), passage.SequenceNumber, passage.CitationLabel).Scan(&commentaryID); err != nil {
			return fmt.Errorf("upsert commentary %d: %w", passage.SequenceNumber, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM commentary_verse WHERE commentary_id=$1`, commentaryID); err != nil {
			return err
		}
		for position, number := range passage.VerseNumbers {
			verseID, ok := verseIDs[number]
			if !ok {
				return fmt.Errorf("BG %d.%s not found", c.Chapter, number)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO commentary_verse (commentary_id,verse_id,relation_type,position) VALUES ($1,$2,'primary',$3)`, commentaryID, verseID, position+1); err != nil {
				return err
			}
		}
	}
	// Remove stale passages if a later parse produces fewer records.
	if _, err := tx.Exec(ctx, `DELETE FROM commentary WHERE source_id=$1 AND chapter_id=$2 AND sequence_number>$3`, commentarySourceID, chapterID, len(c.CommentaryPassages)); err != nil {
		return err
	}

	verseNumbers := make([]string, 0, len(c.Translations))
	for _, item := range c.Translations {
		verseNumbers = append(verseNumbers, item.VerseNumber)
	}
	sort.Strings(verseNumbers)
	if _, err := tx.Exec(ctx, `INSERT INTO data_import (dataset,source_url,source_version,importer_version,chapter_count,verse_count) VALUES ('chinmayananda/holy-geeta',$1,'chapter-1',$2,1,$3)`, sourcePath, ImporterVersion, len(verseNumbers)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}

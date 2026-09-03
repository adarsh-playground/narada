package scripture

type Chapter struct {
	ID                  string `json:"id"`
	Number              int    `json:"number"`
	Title               string `json:"title,omitempty"`
	OriginalTitle       string `json:"original_title,omitempty"`
	TransliteratedTitle string `json:"transliterated_title,omitempty"`
	Meaning             string `json:"meaning,omitempty"`
	Summary             string `json:"summary,omitempty"`
	SummaryHindi        string `json:"summary_hindi,omitempty"`
	VerseCount          int    `json:"verse_count"`
}

type Verse struct {
	ID                      string        `json:"id"`
	Reference               string        `json:"reference"`
	ChapterNumber           int           `json:"chapter_number"`
	VerseNumber             string        `json:"verse_number"`
	SequenceNumber          int           `json:"sequence_number"`
	ScriptureSequenceNumber int           `json:"scripture_sequence_number"`
	OriginalText            string        `json:"original_text"`
	Transliteration         string        `json:"transliteration,omitempty"`
	WordMeanings            string        `json:"word_meanings,omitempty"`
	Translations            []Translation `json:"translations"`
	Commentaries            []Commentary  `json:"commentaries"`
}

type Translation struct {
	Source string `json:"source"`
	Text   string `json:"text"`
}

type Commentary struct {
	Source        string `json:"source"`
	CitationLabel string `json:"citation_label,omitempty"`
	Text          string `json:"text"`
}

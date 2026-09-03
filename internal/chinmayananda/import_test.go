package chinmayananda

import (
	"fmt"
	"testing"
)

func TestLoadChapterOne(t *testing.T) {
	chapter, err := Load("../../data/gita/commentaries/Chinmayananda/chapter_1.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(chapter.Translations); got != 47 {
		t.Fatalf("translation count = %d, want 47", got)
	}
	if got := len(chapter.CommentaryPassages); got == 0 {
		t.Fatal("no commentary passages")
	}
	foundShared := false
	for _, passage := range chapter.CommentaryPassages {
		if passage.CitationLabel == "BG 1.4-6" && len(passage.VerseNumbers) == 3 {
			foundShared = true
		}
	}
	if !foundShared {
		t.Fatal("shared commentary for BG 1.4-6 not found")
	}
}

func TestLoadAllChapters(t *testing.T) {
	totalTranslations := 0
	for chapterNumber := 1; chapterNumber <= 18; chapterNumber++ {
		path := fmt.Sprintf("../../data/gita/commentaries/Chinmayananda/chapter_%d.json", chapterNumber)
		chapter, err := Load(path)
		if err != nil {
			t.Fatalf("chapter %d: %v", chapterNumber, err)
		}
		if chapter.Chapter != chapterNumber {
			t.Fatalf("%s contains chapter %d", path, chapter.Chapter)
		}
		totalTranslations += len(chapter.Translations)
	}
	if totalTranslations != 701 {
		t.Fatalf("translation count = %d, want 701", totalTranslations)
	}
}

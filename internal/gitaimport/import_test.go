package gitaimport

import (
	"testing"
)

func TestLoadRepositoryCorpus(t *testing.T) {
	corpus, err := Load("../../data/gita/chapters.json", "../../data/gita/verse.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(corpus.Chapters); got != 18 {
		t.Fatalf("chapter count = %d, want 18", got)
	}
	if got := len(corpus.Verses); got != 701 {
		t.Fatalf("verse count = %d, want 701", got)
	}
}

func TestValidateRejectsDuplicateVerse(t *testing.T) {
	corpus, err := Load("../../data/gita/chapters.json", "../../data/gita/verse.json")
	if err != nil {
		t.Fatal(err)
	}
	corpus.Verses[1].ChapterNumber = corpus.Verses[0].ChapterNumber
	corpus.Verses[1].VerseNumber = corpus.Verses[0].VerseNumber
	if err := corpus.Validate(); err == nil {
		t.Fatal("Validate() accepted a duplicate verse reference")
	}
}

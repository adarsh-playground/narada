package searchchunk

import (
	"strings"
	"testing"
)

func TestVerseTranslationChunk(t *testing.T) {
	chunk := VerseTranslationChunk("scripture", "verse", "source", "Bhagavad Gita", "BG 2.47", "Swami Chinmayananda", " Thy right is to work only. ")
	if chunk.Kind != "verse_translation" || chunk.CitationLabel != "BG 2.47" {
		t.Fatalf("unexpected metadata: %+v", chunk)
	}
	if chunk.StableKey != "translation:verse:source" {
		t.Fatalf("stable key = %q", chunk.StableKey)
	}
	if !strings.Contains(chunk.Text, "Translation by Swami Chinmayananda") || !strings.HasSuffix(chunk.Text, "Thy right is to work only.") {
		t.Fatalf("unexpected text: %q", chunk.Text)
	}
	if len(chunk.ContentSHA256) != 64 || len(chunk.VerseIDs) != 1 {
		t.Fatalf("invalid derived fields: %+v", chunk)
	}
}

func TestCommentaryChunkPreservesAllVerseLinks(t *testing.T) {
	chunk := CommentaryChunk("scripture", "commentary", "source", "Bhagavad Gita", "BG 1.4-6", "Swami Chinmayananda", "One passage for three verses.", []string{"v4", "v5", "v6"})
	if chunk.StableKey != "commentary:commentary" || len(chunk.VerseIDs) != 3 {
		t.Fatalf("unexpected chunk: %+v", chunk)
	}
	if chunk.CommentaryID != "commentary" {
		t.Fatalf("commentary id = %q", chunk.CommentaryID)
	}
}

package semanticsearch

import "testing"

func TestSortResultsGroupsByKindThenSimilarity(t *testing.T) {
	results := []Result{
		{ChunkID: "translation-high", Kind: KindTranslation, Similarity: 0.95},
		{ChunkID: "commentary-low", Kind: KindCommentary, Similarity: 0.72},
		{ChunkID: "commentary-high", Kind: KindCommentary, Similarity: 0.88},
		{ChunkID: "translation-low", Kind: KindTranslation, Similarity: 0.65},
	}

	sortResults(results)
	want := []string{"commentary-high", "commentary-low", "translation-high", "translation-low"}
	for i := range want {
		if results[i].ChunkID != want[i] {
			t.Fatalf("result %d = %q, want %q", i, results[i].ChunkID, want[i])
		}
	}
}

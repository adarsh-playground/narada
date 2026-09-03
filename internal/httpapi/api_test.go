package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adarsh/narada/internal/askhistory"
	"github.com/adarsh/narada/internal/groundedanswer"
	"github.com/adarsh/narada/internal/scripture"
	"github.com/adarsh/narada/internal/semanticsearch"
	"github.com/labstack/echo/v4"
)

type fakeStore struct {
	chapters []scripture.Chapter
	chapter  scripture.Chapter
	verses   []scripture.Verse
	verse    scripture.Verse
	err      error
}

type fakeSearcher struct {
	results []semanticsearch.Result
	usage   semanticsearch.Usage
	err     error
}

type fakeAnswerer struct {
	answer groundedanswer.Answer
	err    error
}

func (a fakeAnswerer) Generate(context.Context, string, []semanticsearch.Result) (groundedanswer.Answer, error) {
	return a.answer, a.err
}
func (a fakeAnswerer) Model() string { return a.answer.Model }

func (s fakeSearcher) Search(context.Context, string, string, string, int) ([]semanticsearch.Result, error) {
	return s.results, s.err
}
func (s fakeSearcher) SearchWithUsage(context.Context, string, string, string, int) ([]semanticsearch.Result, semanticsearch.Usage, error) {
	return s.results, s.usage, s.err
}

type fakeHistory struct {
	completed bool
	usage     semanticsearch.Usage
}

func (h *fakeHistory) Start(context.Context, string, string) (string, error) {
	return "history-one", nil
}
func (h *fakeHistory) Complete(_ context.Context, _ string, _ groundedanswer.Answer, usage semanticsearch.Usage, _ []semanticsearch.Result, _ time.Duration) error {
	h.completed, h.usage = true, usage
	return nil
}
func (h *fakeHistory) Fail(context.Context, string, error, semanticsearch.Usage, time.Duration) error {
	return nil
}

var _ askhistory.Recorder = (*fakeHistory)(nil)

func (s fakeStore) ListChapters(context.Context, string) ([]scripture.Chapter, error) {
	return s.chapters, s.err
}
func (s fakeStore) GetChapter(context.Context, string, int) (scripture.Chapter, error) {
	return s.chapter, s.err
}
func (s fakeStore) ListVerses(context.Context, string, int) ([]scripture.Verse, error) {
	return s.verses, s.err
}
func (s fakeStore) GetVerse(context.Context, string, int, string) (scripture.Verse, error) {
	return s.verse, s.err
}
func (s fakeStore) RandomVerse(context.Context) (scripture.Verse, error) {
	return s.verse, s.err
}

func TestListChapters(t *testing.T) {
	e := New(fakeStore{chapters: []scripture.Chapter{{Number: 1, Title: "Arjuna Visada Yoga", VerseCount: 47}}})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"verse_count":47`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestGetVerse(t *testing.T) {
	e := New(fakeStore{verse: scripture.Verse{
		Reference: "BG 2.47", ChapterNumber: 2, VerseNumber: "47", OriginalText: "कर्मण्येवाधिकारस्ते",
		Translations: []scripture.Translation{{Source: "Swami Chinmayananda", Text: "Thy right is to work only."}},
		Commentaries: []scripture.Commentary{{Source: "Swami Chinmayananda", CitationLabel: "BG 2.47", Text: "The verse advises right action."}},
	}})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters/2/verses/47")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"reference":"BG 2.47"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"translations":[{"source":"Swami Chinmayananda"`) ||
		!strings.Contains(recorder.Body.String(), `"commentaries":[{"source":"Swami Chinmayananda"`) {
		t.Fatalf("translation or commentary missing: %s", recorder.Body.String())
	}
}

func TestRandomVerse(t *testing.T) {
	e := New(fakeStore{verse: scripture.Verse{Reference: "BG 18.78", ChapterNumber: 18, VerseNumber: "78"}})
	recorder := serve(e, http.MethodGet, "/api/v1/verses/random")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"reference":"BG 18.78"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInvalidChapter(t *testing.T) {
	e := New(fakeStore{})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters/not-a-number")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestNotFound(t *testing.T) {
	e := New(fakeStore{err: scripture.ErrNotFound})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters/99/verses")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestInternalErrorDoesNotLeakDetails(t *testing.T) {
	e := New(fakeStore{err: errors.New("database password leaked")})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "password") {
		t.Fatalf("response leaked internal error: %s", recorder.Body.String())
	}
}

func TestRequestIDIsReturned(t *testing.T) {
	e := New(fakeStore{chapters: []scripture.Chapter{}})
	recorder := serve(e, http.MethodGet, "/api/v1/scriptures/BG/chapters")
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID response header is empty")
	}
}

func TestOversizedRequestBodyIsRejected(t *testing.T) {
	e := NewWithAnswerer(fakeStore{}, fakeSearcher{}, fakeAnswerer{})
	body := `{"question":"` + strings.Repeat("x", 17<<10) + `"}`
	recorder := serveBody(e, http.MethodPost, "/api/v1/ask", body)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
}

func TestPanicsAreRecovered(t *testing.T) {
	e := New(fakeStore{})
	e.GET("/panic", func(echo.Context) error {
		panic("test panic")
	})
	recorder := serve(e, http.MethodGet, "/panic")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestSemanticSearch(t *testing.T) {
	e := New(fakeStore{}, fakeSearcher{results: []semanticsearch.Result{{
		Kind: semanticsearch.KindCommentary, CitationLabel: "BG 2.47", Source: "Swami Chinmayananda",
		Text: "Act without attachment.", VerseRefs: []string{"BG 2.47"}, Similarity: 0.81,
	}}})
	recorder := serve(e, http.MethodGet, "/api/v1/search?q=how+should+I+work%3F&limit=5")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"citation_label":"BG 2.47"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSemanticSearchValidatesInput(t *testing.T) {
	e := New(fakeStore{}, fakeSearcher{})
	for _, target := range []string{"/api/v1/search", "/api/v1/search?q=duty&limit=21", "/api/v1/search?q=duty&kind=verse"} {
		if recorder := serve(e, http.MethodGet, target); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d, want 400", target, recorder.Code)
		}
	}
}

func TestAskReturnsGroundedAnswerAndEvidence(t *testing.T) {
	searcher := fakeSearcher{usage: semanticsearch.Usage{EmbeddingModel: "text-embedding-3-small", EmbeddingInputTokens: 9}, results: []semanticsearch.Result{{ChunkID: "one", Kind: semanticsearch.KindCommentary, CitationLabel: "BG 2.47", Text: "Work without attachment."}}}
	answerer := fakeAnswerer{answer: groundedanswer.Answer{Text: "Focus on the work [BG 2.47].", Model: "gpt-test", InputTokens: 100, OutputTokens: 10}}
	history := &fakeHistory{}
	e := NewWithAnswererAndHistory(fakeStore{}, searcher, answerer, history)
	recorder := serveBody(e, http.MethodPost, "/api/v1/ask", `{"question":"How should I work?"}`)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"text":"Focus on the work [BG 2.47]."`) || !strings.Contains(recorder.Body.String(), `"citation_label":"BG 2.47"`) {
		t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !history.completed || history.usage.EmbeddingInputTokens != 9 {
		t.Fatalf("history was not completed with usage: %+v", history)
	}
}

func TestAskGlobalRateLimit(t *testing.T) {
	searcher := fakeSearcher{results: []semanticsearch.Result{{Kind: semanticsearch.KindCommentary, Text: "Work without attachment."}}}
	answerer := fakeAnswerer{answer: groundedanswer.Answer{Text: "Focus on the work.", Model: "gpt-test"}}
	e := NewWithAnswerer(fakeStore{}, searcher, answerer)
	for requestNumber := 1; requestNumber <= 11; requestNumber++ {
		recorder := serveBody(e, http.MethodPost, "/api/v1/ask", `{"question":"How should I work?"}`)
		if requestNumber <= 10 && recorder.Code != http.StatusOK {
			t.Fatalf("request %d: status=%d, want 200", requestNumber, recorder.Code)
		}
		if requestNumber == 11 {
			if recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("request 11: status=%d, want 429", recorder.Code)
			}
			if recorder.Header().Get("Retry-After") != "60" {
				t.Fatalf("Retry-After = %q, want 60", recorder.Header().Get("Retry-After"))
			}
		}
	}
}

func serve(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	return serveBody(handler, method, target, "")
}

func serveBody(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

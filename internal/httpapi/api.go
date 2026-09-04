package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adarsh/narada/internal/askhistory"
	"github.com/adarsh/narada/internal/groundedanswer"
	"github.com/adarsh/narada/internal/scripture"
	"github.com/adarsh/narada/internal/semanticsearch"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type API struct {
	store    scripture.Store
	searcher semanticsearch.Searcher
	answerer groundedanswer.Generator
	history  askhistory.Recorder
}

func New(store scripture.Store, searchers ...semanticsearch.Searcher) *echo.Echo {
	api := &API{store: store}
	if len(searchers) > 0 {
		api.searcher = searchers[0]
	}
	return routes(api)
}

func NewWithAnswerer(store scripture.Store, searcher semanticsearch.Searcher, answerer groundedanswer.Generator) *echo.Echo {
	return routes(&API{store: store, searcher: searcher, answerer: answerer})
}

func NewWithAnswererAndHistory(store scripture.Store, searcher semanticsearch.Searcher, answerer groundedanswer.Generator, history askhistory.Recorder) *echo.Echo {
	return routes(&API{store: store, searcher: searcher, answerer: answerer, history: history})
}

func routes(api *API) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("16K"))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType},
	}))

	e.GET("/health", api.health)
	e.GET("/api/v1/verses/random", api.randomVerse)
	e.GET("/api/v1/search", api.search)
	askLimiter := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(10.0 / 60.0),
		Burst:     10,
		ExpiresIn: 10 * time.Minute,
	})
	e.POST("/api/v1/ask", api.ask, middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: askLimiter,
		IdentifierExtractor: func(echo.Context) (string, error) {
			return "global", nil
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			c.Response().Header().Set("Retry-After", "60")
			return c.JSON(http.StatusTooManyRequests, errorResponse{Error: "too many questions; please wait a minute and try again"})
		},
	}))
	e.GET("/api/v1/scriptures/:scripture/chapters", api.listChapters)
	e.GET("/api/v1/scriptures/:scripture/chapters/:chapter", api.getChapter)
	e.GET("/api/v1/scriptures/:scripture/chapters/:chapter/verses", api.listVerses)
	e.GET("/api/v1/scriptures/:scripture/chapters/:chapter/verses/:verse", api.getVerse)
	return e
}

type askRequest struct {
	Question  string `json:"question"`
	Scripture string `json:"scripture"`
}

func (a *API) ask(c echo.Context) error {
	startedAt := time.Now()
	if a.searcher == nil || a.answerer == nil {
		return c.JSON(http.StatusServiceUnavailable, errorResponse{Error: "grounded answers are unavailable; OPENAI_API_KEY is not configured"})
	}
	var request askRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "request body must be valid JSON"})
	}
	request.Question = strings.TrimSpace(request.Question)
	if request.Question == "" {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "question is required"})
	}
	if len(request.Question) > 1000 {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "question must be 1000 characters or fewer"})
	}
	if strings.TrimSpace(request.Scripture) == "" {
		request.Scripture = "BG"
	}
	ctx := c.Request().Context()
	historyID := ""
	if a.history != nil {
		var err error
		historyID, err = a.history.Start(ctx, request.Question, request.Scripture)
		if err != nil {
			return respondError(c, err)
		}
	}
	if cache, ok := a.history.(askhistory.CacheReader); ok {
		cached, found, err := cache.FindCompleted(ctx, request.Question, request.Scripture)
		if err != nil {
			a.recordFailure(ctx, historyID, err, semanticsearch.Usage{}, time.Since(startedAt))
			return respondError(c, err)
		}
		if found {
			cached.Answer.InputTokens = 0
			cached.Answer.OutputTokens = 0
			if err := a.history.Complete(context.WithoutCancel(ctx), historyID, cached.Answer,
				semanticsearch.Usage{}, cached.Evidence, time.Since(startedAt)); err != nil {
				return respondError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"question": request.Question, "answer": cached.Answer, "results": cached.Evidence,
			})
		}
	}
	usage := semanticsearch.Usage{}
	var results []semanticsearch.Result
	var err error
	if searcher, ok := a.searcher.(semanticsearch.MeteredSearcher); ok {
		results, usage, err = searcher.SearchWithUsage(ctx, request.Question, request.Scripture, "", 8)
	} else {
		results, err = a.searcher.Search(ctx, request.Question, request.Scripture, "", 8)
	}
	if err != nil {
		a.recordFailure(ctx, historyID, err, usage, time.Since(startedAt))
		return respondError(c, err)
	}
	if len(results) == 0 {
		if a.history != nil {
			if err := a.history.Complete(context.WithoutCancel(ctx), historyID, groundedanswer.Answer{Model: a.answerer.Model()}, usage, results, time.Since(startedAt)); err != nil {
				return respondError(c, err)
			}
		}
		return c.JSON(http.StatusOK, map[string]any{"question": request.Question, "answer": nil, "results": results})
	}
	answer, err := a.answerer.Generate(ctx, request.Question, results)
	if err != nil {
		a.recordFailure(ctx, historyID, err, usage, time.Since(startedAt))
		return respondError(c, err)
	}
	if a.history != nil {
		if err := a.history.Complete(context.WithoutCancel(ctx), historyID, answer, usage, results, time.Since(startedAt)); err != nil {
			return respondError(c, err)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"question": request.Question, "answer": answer, "results": results})
}

func (a *API) recordFailure(ctx context.Context, historyID string, failure error, usage semanticsearch.Usage, duration time.Duration) {
	if a.history == nil || historyID == "" {
		return
	}
	if err := a.history.Fail(context.WithoutCancel(ctx), historyID, failure, usage, duration); err != nil {
		log.Printf("record ask failure: %v", err)
	}
}

func (a *API) search(c echo.Context) error {
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "q is required"})
	}
	limit := 8
	if value := c.QueryParam("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 20 {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: "limit must be between 1 and 20"})
		}
		limit = parsed
	}
	kind := strings.TrimSpace(c.QueryParam("kind"))
	if kind != "" && kind != semanticsearch.KindTranslation && kind != semanticsearch.KindCommentary {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "kind must be verse_translation or commentary"})
	}
	if a.searcher == nil {
		return c.JSON(http.StatusServiceUnavailable, errorResponse{Error: "semantic search is unavailable; OPENAI_API_KEY is not configured"})
	}
	results, err := a.searcher.Search(c.Request().Context(), query, strings.TrimSpace(c.QueryParam("scripture")), kind, limit)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"query": query, "results": results})
}

func (a *API) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) randomVerse(c echo.Context) error {
	verse, err := a.store.RandomVerse(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, verse)
}

func (a *API) listChapters(c echo.Context) error {
	chapters, err := a.store.ListChapters(c.Request().Context(), c.Param("scripture"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"chapters": chapters})
}

func (a *API) getChapter(c echo.Context) error {
	chapterNumber, err := positiveInteger(c.Param("chapter"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "chapter must be a positive integer"})
	}
	chapter, err := a.store.GetChapter(c.Request().Context(), c.Param("scripture"), chapterNumber)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, chapter)
}

func (a *API) listVerses(c echo.Context) error {
	chapterNumber, err := positiveInteger(c.Param("chapter"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "chapter must be a positive integer"})
	}
	verses, err := a.store.ListVerses(c.Request().Context(), c.Param("scripture"), chapterNumber)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"verses": verses})
}

func (a *API) getVerse(c echo.Context) error {
	chapterNumber, err := positiveInteger(c.Param("chapter"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "chapter must be a positive integer"})
	}
	verseNumber := strings.TrimSpace(c.Param("verse"))
	if verseNumber == "" {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "verse is required"})
	}
	verse, err := a.store.GetVerse(c.Request().Context(), c.Param("scripture"), chapterNumber, verseNumber)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(http.StatusOK, verse)
}

func positiveInteger(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 {
		return 0, errors.New("not a positive integer")
	}
	return number, nil
}

type errorResponse struct {
	Error string `json:"error"`
}

func respondError(c echo.Context, err error) error {
	if errors.Is(err, scripture.ErrNotFound) {
		return c.JSON(http.StatusNotFound, errorResponse{Error: "scripture record not found"})
	}
	log.Printf("request_id=%s request failed: %v", c.Response().Header().Get(echo.HeaderXRequestID), err)
	return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

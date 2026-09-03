package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adarsh/narada/internal/askhistory"
	"github.com/adarsh/narada/internal/database"
	"github.com/adarsh/narada/internal/embedding"
	"github.com/adarsh/narada/internal/groundedanswer"
	"github.com/adarsh/narada/internal/httpapi"
	"github.com/adarsh/narada/internal/scripture"
	"github.com/adarsh/narada/internal/semanticsearch"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.OpenPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var searcher semanticsearch.Searcher
	var answerer groundedanswer.Generator
	var history askhistory.Recorder
	if os.Getenv("OPENAI_API_KEY") != "" {
		provider, err := embedding.NewOpenAI(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("EMBEDDING_MODEL"), nil)
		if err != nil {
			log.Fatal(err)
		}
		answerer, err = groundedanswer.NewOpenAI(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("ANSWER_MODEL"), nil)
		if err != nil {
			log.Fatal(err)
		}
		history, err = askhistory.New(pool, provider.Model(), answerer.Model(), groundedanswer.PromptVersion)
		if err != nil {
			log.Fatal(err)
		}
		provider.SetRetryNotifier(func(delay time.Duration, err error) {
			log.Printf("%v; retrying automatically in %s", err, delay.Round(time.Second))
		})
		searcher = semanticsearch.New(pool, provider)
	} else {
		log.Printf("OPENAI_API_KEY is not set; semantic search endpoint will be unavailable")
	}
	server := httpapi.NewWithAnswererAndHistory(scripture.NewPostgresStore(pool), searcher, answerer, history)
	server.Server.ReadHeaderTimeout = 5 * time.Second
	server.Server.ReadTimeout = 15 * time.Second
	server.Server.WriteTimeout = 120 * time.Second
	server.Server.IdleTimeout = 60 * time.Second
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := server.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("API stopped unexpectedly: %v", err)
			stop()
		}
	}()
	log.Printf("Narada API listening on http://localhost:%s", port)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("API shutdown failed: %v", err)
	}
}

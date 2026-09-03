package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adarsh/narada/internal/database"
	"github.com/adarsh/narada/internal/embedding"
)

func main() {
	batchSize := flag.Int("batch-size", 32, "embedding inputs per request")
	limit := flag.Int("limit", 0, "maximum chunks to embed; zero means all")
	flag.Parse()
	provider, err := embedding.NewOpenAI(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("EMBEDDING_MODEL"), nil)
	if err != nil {
		log.Fatal(err)
	}
	provider.SetRetryNotifier(func(delay time.Duration, err error) {
		log.Printf("%v; retrying automatically in %s", err, delay.Round(time.Second))
	})
	ctx := context.Background()
	conn, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	count, err := embedding.Generate(ctx, conn, provider, *batchSize, *limit)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("generated %d embeddings with %s (%d dimensions)\n", count, provider.Model(), provider.Dimensions())
}

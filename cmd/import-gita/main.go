package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adarsh/narada/internal/database"
	"github.com/adarsh/narada/internal/gitaimport"
)

func main() {
	chaptersPath := flag.String("chapters", "data/gita/chapters.json", "chapter JSON file")
	versesPath := flag.String("verses", "data/gita/verse.json", "verse JSON file")
	sourceVersion := flag.String("source-version", "unknown", "upstream commit or version")
	validateOnly := flag.Bool("validate-only", false, "validate files without writing to PostgreSQL")
	flag.Parse()

	corpus, err := gitaimport.Load(*chaptersPath, *versesPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("validated %d chapters and %d verses\n", len(corpus.Chapters), len(corpus.Verses))
	if *validateOnly {
		return
	}

	ctx := context.Background()
	conn, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	if err := corpus.Import(ctx, conn, *sourceVersion); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Gita import completed")
}

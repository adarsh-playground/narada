package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adarsh/narada/internal/database"
	"github.com/adarsh/narada/internal/searchchunk"
)

func main() {
	shortName := flag.String("scripture", "BG", "scripture short name")
	flag.Parse()
	ctx := context.Background()
	conn, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	count, err := searchchunk.Build(ctx, conn, *shortName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("built %d search chunks for %s\n", count, *shortName)
}

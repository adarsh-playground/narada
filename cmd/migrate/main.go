package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adarsh/narada/internal/database"
)

func main() {
	directory := flag.String("dir", "migrations", "migration directory")
	flag.Parse()

	ctx := context.Background()
	conn, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	if err := database.Migrate(ctx, conn, *directory); err != nil {
		log.Fatal(err)
	}
	fmt.Println("database migrations are up to date")
}

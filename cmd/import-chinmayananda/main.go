package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/adarsh/narada/internal/chinmayananda"
	"github.com/adarsh/narada/internal/database"
)

func main() {
	input := flag.String("input", "", "one parsed chapter JSON (defaults to all chapter_*.json files)")
	inputDir := flag.String("input-dir", "data/gita/commentaries/Chinmayananda", "directory containing parsed chapter JSON files")
	validateOnly := flag.Bool("validate-only", false, "validate without writing to PostgreSQL")
	flag.Parse()
	paths := []string{*input}
	if *input == "" {
		var err error
		paths, err = filepath.Glob(filepath.Join(*inputDir, "chapter_*.json"))
		if err != nil || len(paths) == 0 {
			log.Fatalf("find parsed chapters: %v", err)
		}
	}
	type loadedChapter struct {
		path    string
		chapter chinmayananda.Chapter
	}
	loaded := make([]loadedChapter, 0, len(paths))
	for _, path := range paths {
		chapter, err := chinmayananda.Load(path)
		if err != nil {
			log.Fatal(err)
		}
		loaded = append(loaded, loadedChapter{path, chapter})
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].chapter.Chapter < loaded[j].chapter.Chapter })
	for _, item := range loaded {
		fmt.Printf("validated chapter %d: %d translations, %d commentary passages\n", item.chapter.Chapter, len(item.chapter.Translations), len(item.chapter.CommentaryPassages))
	}
	if *validateOnly {
		return
	}
	ctx := context.Background()
	conn, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, item := range loaded {
		if err := item.chapter.Import(ctx, conn, item.path); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Chinmayananda import completed for %d chapters\n", len(loaded))
}

package main

import (
	"context"
	"log"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/parser"
	"github.com/elijahthis/baby-crawler/internal/storage"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password", // no password set
		DB:       0,
	})

	fr := frontier.NewRedisFrontier(rdb)

	htmlParser := parser.NewHTMLParser()

	store, err := storage.NewS3Storage(context.Background(), "crawled-data", "http://localhost:9000", "admin", "password")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// setup coordinator
	coord := parser.NewService(fr, store, htmlParser, 10)
	log.Println("Starting parser")
	coord.Run(context.Background())
}

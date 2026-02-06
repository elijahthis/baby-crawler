package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/elijahthis/baby-crawler/internal/crawler"
	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/limiter"
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
	log.Println("Frontier created")

	baseFetcher := crawler.NewWebFetcher("BabyCrawler/1.0", 5*time.Second)
	fetcher := &crawler.RetryFetcher{
		Base:    baseFetcher,
		Retries: 3,
	}

	parser := parser.NewHTMLParser()
	log.Println("Fetcher created")

	redisLimiter := limiter.NewRedisRateLimiter(rdb, 1*time.Second)

	store, err := storage.NewS3Storage(context.Background(), "crawled-data", "http://localhost:9000", "admin", "password")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	if err := fr.Push(context.Background(), []string{"https://grpc.io/docs/"}, 0); err != nil {
		fmt.Printf("Error: %s", err.Error())
	}

	// setup coordinator
	coord := crawler.NewCoordinator(fr, fetcher, parser, redisLimiter, store, 10)
	log.Println("Starting fetch")
	coord.Run(context.Background())
}

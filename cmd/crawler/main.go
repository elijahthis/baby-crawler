package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/elijahthis/baby-crawler/internal/crawler"
	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/limiter"
	"github.com/elijahthis/baby-crawler/internal/robots"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/elijahthis/baby-crawler/internal/storage"
	"github.com/redis/go-redis/v9"
)

var userAgent = "BabyCrawler/1.0"

func main() {
	shared.InitLogger("crawler")

	log.Info().Msg("Starting Fetcher Service...")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password", // no password set
		DB:       0,
	})

	fr := frontier.NewRedisFrontier(rdb)
	log.Info().Msg("Frontier created")

	baseFetcher := crawler.NewWebFetcher(userAgent, 5*time.Second)
	fetcher := &crawler.RetryFetcher{
		Base:    baseFetcher,
		Retries: 3,
	}

	redisLimiter := limiter.NewRedisRateLimiter(rdb, 1*time.Second)

	store, err := storage.NewS3Storage(context.Background(), "crawled-data", "http://localhost:9000", "admin", "password")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	robotChecker := robots.NewRobotsChecker(userAgent, 5*time.Second)

	if err := fr.Push(context.Background(), []string{"https://lanre.wtf/"}, 0); err != nil {
		log.Error().Err(err).Msg("Frontier Push Error")
	}

	// setup coordinator
	coord := crawler.NewCoordinator(fr, fetcher, redisLimiter, store, robotChecker, 10)
	log.Info().Msg("Starting fetch")
	coord.Run(context.Background())
}

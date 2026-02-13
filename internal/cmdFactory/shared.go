package cmdfactory

import (
	"context"
	"time"

	"github.com/elijahthis/baby-crawler/internal/crawler"
	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/limiter"
	"github.com/elijahthis/baby-crawler/internal/robots"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/elijahthis/baby-crawler/internal/storage"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// MinIO / S3
	S3Endpoint string
	S3Bucket   string
	S3User     string
	S3Password string

	// Crawler Specific
	SeedURLs           []string // Only used by Crawler
	CrawlerWorkerCount int
	CrawlerMetricsPort int

	// Parser Specific
	ParserWorkerCount int
	ParserMetricsPort int
}

func newFetcher(f *commonFactory) shared.Fetcher {
	baseFetcher := crawler.NewWebFetcher(f.userAgent, 5*time.Second)
	fetcher := &crawler.RetryFetcher{
		Base:    baseFetcher,
		Retries: 3,
	}
	log.Info().Msg("Starting Fetcher Service...")
	return fetcher
}

func newFrontier(f *commonFactory) frontier.Frontier {
	fr := frontier.NewRedisFrontier(f.RDB)
	log.Info().Msg("Frontier created")
	return fr
}

func newRateLimiter(f *commonFactory) shared.RateLimiter {
	redisLimiter := limiter.NewRedisRateLimiter(f.RDB)
	return redisLimiter
}

func newRobot(f *commonFactory) *robots.RobotsChecker {
	robotChecker := robots.NewRobotsChecker(f.userAgent, 5*time.Second)
	return robotChecker
}

func newStorage(cfg *Config) shared.Storage {
	store, err := storage.NewS3Storage(context.Background(), cfg.S3Bucket, cfg.S3Endpoint, cfg.S3User, cfg.S3Password)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}
	return store
}

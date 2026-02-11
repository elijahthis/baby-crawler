package cmdfactory

import (
	"context"
	"strconv"

	"github.com/elijahthis/baby-crawler/internal/crawler"
	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/metrics"
	"github.com/elijahthis/baby-crawler/internal/robots"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/redis/go-redis/v9"
)

type commonFactory struct {
	RDB         *redis.Client
	userAgent   string
	workerCount int
	Metrics     *metrics.PrometheusMetrics
}

type crawlerFactory struct {
	*commonFactory
	Frontier    frontier.Frontier
	Fetcher     shared.Fetcher
	RateLimiter shared.RateLimiter
	Store       shared.Storage
	Robots      *robots.RobotsChecker
	Coordinator *crawler.Coordinator
}

func CrawlerNew(cfg *Config) *crawlerFactory {

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword, // no password set
		DB:       cfg.RedisDB,
	})
	met := metrics.NewMetrics()
	go metrics.StartNewMetricsServer(":" + strconv.Itoa(cfg.CrawlerMetricsPort))

	queuesToWatch := map[string]string{
		"frontier":     "crawler:queue",
		"parser_queue": "crawler:parser_queue",
		"dlq":          "crawler:dlq",
	}

	go met.MonitorQueueDepth(context.Background(), rdb, queuesToWatch)

	f := &crawlerFactory{
		commonFactory: &commonFactory{
			userAgent:   "BabyCrawler/1.0",
			RDB:         rdb,
			workerCount: cfg.CrawlerWorkerCount,
		},
	}

	f.Frontier = newFrontier(f.commonFactory)
	f.Fetcher = newFetcher(f.commonFactory)
	f.RateLimiter = newRateLimiter(f.commonFactory)
	f.Robots = newRobot(f.commonFactory)
	f.Metrics = met

	f.Store = newStorage(cfg)
	f.Coordinator = newCoordinator(f)

	return f
}

func newCoordinator(f *crawlerFactory) *crawler.Coordinator {
	coord := crawler.NewCoordinator(f.Frontier, f.Fetcher, f.RateLimiter, f.Store, f.Robots, f.workerCount, f.Metrics)
	return coord
}

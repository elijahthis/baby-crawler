package cmdfactory

import (
	"github.com/elijahthis/baby-crawler/internal/crawler"
	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/robots"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/redis/go-redis/v9"
)

type commonFactory struct {
	RDB         *redis.Client
	userAgent   string
	workerCount int
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

	f.Store = newStorage(cfg)
	f.Coordinator = newCoordinator(f)

	return f
}

func newCoordinator(f *crawlerFactory) *crawler.Coordinator {
	coord := crawler.NewCoordinator(f.Frontier, f.Fetcher, f.RateLimiter, f.Store, f.Robots, f.workerCount)
	return coord
}
